package namespace

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"
)

const MaxResponseBytes = 8 << 20
const TransformTimeout = 5 * time.Second
const maxElements = 50000
const s3XMLNamespace = "http://s3.amazonaws.com/doc/2006-03-01/"

var ErrResponse = errors.New("invalid upstream listing response")
var ErrTooLarge = errors.New("listing response exceeds transformation limit")

type element struct {
	name     string
	text     string
	children []*element
}

func (m Mapping) Transform(ctx context.Context, body io.ReadCloser, bucket string, query url.Values) (data []byte, err error) {
	return m.transform(ctx, body, bucket, query, TransformTimeout)
}

func (m Mapping) transform(ctx context.Context, body io.ReadCloser, bucket string, query url.Values, timeout time.Duration) (data []byte, err error) {
	if body == nil {
		return nil, ErrResponse
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var once sync.Once
	var closeErr error
	closeBody := func() { once.Do(func() { closeErr = body.Close() }) }
	stop := context.AfterFunc(ctx, closeBody)
	defer func() {
		stop()
		closeBody()
		err = errors.Join(err, closeErr, ctx.Err())
		if err != nil {
			data = nil
		}
	}()
	input, err := io.ReadAll(io.LimitReader(contextReader{ctx, body}, MaxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(input) > MaxResponseBytes {
		return nil, ErrTooLarge
	}
	root, err := parseResult(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := m.translate(root, bucket, query); err != nil {
		return nil, err
	}
	buf := &boundedBuffer{}
	enc := xml.NewEncoder(buf)
	if err := encodeElement(ctx, enc, root, true); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

type boundedBuffer struct{ bytes.Buffer }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > MaxResponseBytes-b.Len() {
		return 0, ErrTooLarge
	}
	return b.Buffer.Write(p)
}

var children = map[string]map[string]bool{
	"ListBucketResult": {"Name": false, "Prefix": false, "StartAfter": false, "Delimiter": false, "EncodingType": false, "MaxKeys": false, "KeyCount": false, "IsTruncated": false, "ContinuationToken": false, "NextContinuationToken": false, "Contents": true, "CommonPrefixes": true},
	"Contents":         {"Key": false, "LastModified": false, "ETag": false, "Size": false, "StorageClass": false, "Owner": false, "ChecksumAlgorithm": true, "ChecksumType": false, "RestoreStatus": false},
	"CommonPrefixes":   {"Prefix": false},
	"Owner":            {"ID": false, "DisplayName": false},
	"RestoreStatus":    {"IsRestoreInProgress": false, "RestoreExpiryDate": false},
}

func parseResult(ctx context.Context, input []byte) (*element, error) {
	dec := xml.NewDecoder(bytes.NewReader(input))
	var root *element
	var stack []*element
	var ns string
	var leafText strings.Builder
	count := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrResponse
		}
		switch t := tok.(type) {
		case xml.StartElement:
			count++
			if count > maxElements {
				return nil, ErrTooLarge
			}
			if len(stack) >= 4 {
				return nil, ErrResponse
			}
			for _, attr := range t.Attr {
				if attr.Name.Local != "xmlns" && attr.Name.Space != "xmlns" {
					return nil, ErrResponse
				}
			}
			n := &element{name: t.Name.Local}
			if len(stack) == 0 {
				if root != nil || n.name != "ListBucketResult" || (t.Name.Space != "" && t.Name.Space != s3XMLNamespace) {
					return nil, ErrResponse
				}
				root, ns = n, t.Name.Space
			} else {
				parent := stack[len(stack)-1]
				repeated, ok := children[parent.name][n.name]
				if !ok || t.Name.Space != ns || (!repeated && parent.child(n.name) != nil) {
					return nil, ErrResponse
				}
				parent.children = append(parent.children, n)
			}
			stack = append(stack, n)
		case xml.EndElement:
			n := stack[len(stack)-1]
			if children[n.name] == nil {
				n.text = leafText.String()
				leafText.Reset()
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 || children[stack[len(stack)-1].name] != nil {
				if strings.TrimSpace(string(t)) != "" {
					return nil, ErrResponse
				}
			} else {
				leafText.Write(t)
			}
		case xml.Directive:
			return nil, ErrResponse
		case xml.ProcInst:
			if t.Target != "xml" || root != nil {
				return nil, ErrResponse
			}
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, ErrResponse
	}
	return root, nil
}

func (e *element) child(name string) *element {
	for _, child := range e.children {
		if child.name == name {
			return child
		}
	}
	return nil
}

func (e *element) value(name string) string {
	if child := e.child(name); child != nil {
		return child.text
	}
	return ""
}

func (m Mapping) translate(root *element, bucket string, query url.Values) error {
	mapped, err := m.Query(query)
	if err != nil {
		return err
	}
	if root.value("Name") != bucket || root.child("Prefix") == nil || root.value("EncodingType") != query.Get("encoding-type") {
		return ErrResponse
	}
	if truncated := root.value("IsTruncated"); (truncated != "true" && truncated != "false") || (truncated == "true" && root.value("NextContinuationToken") == "") {
		return ErrResponse
	}
	encoded := query.Get("encoding-type") == "url"
	decode := func(s string) (string, error) {
		if encoded {
			return url.QueryUnescape(s)
		}
		return s, nil
	}
	encode := func(s string) string {
		if encoded {
			return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
		}
		return s
	}
	root.child("Name").text = m.VisibleBucket
	for _, name := range []string{"Prefix", "StartAfter", "Delimiter"} {
		if field := root.child(name); field != nil {
			value, err := decode(field.text)
			key := map[string]string{"Prefix": "prefix", "StartAfter": "start-after", "Delimiter": "delimiter"}[name]
			matches := (err == nil && value == mapped.Get(key)) || field.text == mapped.Get(key)
			if !matches || (name == "StartAfter" && !query.Has(key)) {
				return ErrResponse
			}
			field.text = encode(query.Get(key))
		}
	}
	if field := root.child("ContinuationToken"); field != nil && field.text != query.Get("continuation-token") {
		return ErrResponse
	}
	for _, entry := range root.children {
		var field *element
		switch entry.name {
		case "Contents":
			field = entry.child("Key")
		case "CommonPrefixes":
			field = entry.child("Prefix")
		default:
			continue
		}
		if field == nil {
			return ErrResponse
		}
		value, err := decode(field.text)
		if err != nil || !strings.HasPrefix(value, mapped.Get("prefix")) || value == m.Prefix {
			return ErrResponse
		}
		field.text = encode(strings.TrimPrefix(value, m.Prefix))
	}
	return nil
}

func encodeElement(ctx context.Context, enc *xml.Encoder, e *element, root bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	start := xml.StartElement{Name: xml.Name{Local: e.name}}
	if root {
		start.Attr = []xml.Attr{{Name: xml.Name{Local: "xmlns"}, Value: s3XMLNamespace}}
	}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	if err := enc.EncodeToken(xml.CharData(e.text)); err != nil {
		return err
	}
	for _, child := range e.children {
		if err := encodeElement(ctx, enc, child, false); err != nil {
			return err
		}
	}
	return enc.EncodeToken(start.End())
}
