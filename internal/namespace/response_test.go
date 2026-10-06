package namespace

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const validXML = `<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>store</Name><Prefix>tenant/</Prefix><IsTruncated>false</IsTruncated><Contents><Key>tenant/a</Key><ETag>etag</ETag></Contents></ListBucketResult>`

type trackedBody struct {
	io.Reader
	closed atomic.Int32
	err    error
}

func (b *trackedBody) Close() error { b.closed.Add(1); return b.err }

func TestTransformRejectsInvalidResponsesAndCloses(t *testing.T) {
	for name, input := range map[string]string{
		"malformed":         validXML[:len(validXML)-1],
		"sibling":           strings.Replace(validXML, "tenant/a", "tenant-other/secret", 1),
		"marker":            strings.Replace(validXML, "tenant/a", "tenant/", 1),
		"common sibling":    strings.Replace(validXML, "<Contents><Key>tenant/a</Key><ETag>etag</ETag></Contents>", "<CommonPrefixes><Prefix>other/</Prefix></CommonPrefixes>", 1),
		"wrong name":        strings.Replace(validXML, "<Name>store", "<Name>other", 1),
		"wrong prefix":      strings.Replace(validXML, "<Prefix>tenant/", "<Prefix>other/", 1),
		"missing key":       strings.Replace(validXML, "<Key>tenant/a</Key>", "", 1),
		"duplicate key":     strings.Replace(validXML, "</Key>", "</Key><Key>tenant/b</Key>", 1),
		"nested key":        strings.Replace(validXML, "tenant/a", "<Key>tenant/a</Key>", 1),
		"unknown field":     strings.Replace(validXML, "<ETag>etag</ETag>", "<Other>secret</Other>", 1),
		"namespace":         strings.Replace(validXML, "<Key>", `<Key xmlns="other">`, 1),
		"second root":       validXML + validXML,
		"trailing text":     validXML + "bad",
		"doctype":           "<!DOCTYPE x>" + validXML,
		"missing token":     strings.Replace(validXML, "false", "true", 1),
		"bad bool":          strings.Replace(validXML, "false", "maybe", 1),
		"encoding mismatch": strings.Replace(validXML, "</Name>", "</Name><EncodingType>url</EncodingType>", 1),
	} {
		t.Run(name, func(t *testing.T) {
			body := &trackedBody{Reader: strings.NewReader(input)}
			data, err := (Mapping{VisibleBucket: "visible", Prefix: "tenant/"}).Transform(context.Background(), body, "store", nil)
			if !errors.Is(err, ErrResponse) || data != nil || body.closed.Load() != 1 {
				t.Fatalf("data=%q err=%v closes=%d", data, err, body.closed.Load())
			}
		})
	}
}

func TestTransformBounds(t *testing.T) {
	for _, tt := range []struct {
		name, input string
		want        error
	}{
		{"exact input limit", validXML + strings.Repeat(" ", MaxResponseBytes-len(validXML)), nil},
		{"input limit plus one", validXML + strings.Repeat(" ", MaxResponseBytes-len(validXML)+1), ErrTooLarge},
		{"element limit", strings.Replace(validXML, "</ListBucketResult>", strings.Repeat("<Contents><Key>tenant/a</Key></Contents>", maxElements/2)+"</ListBucketResult>", 1), ErrTooLarge},
		{"output expansion", strings.Replace(validXML, "etag", strings.Repeat("\"", MaxResponseBytes/5), 1), ErrTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &trackedBody{Reader: strings.NewReader(tt.input)}
			_, err := (Mapping{VisibleBucket: "visible", Prefix: "tenant/"}).transform(context.Background(), body, "store", nil, 30*time.Second)
			if !errors.Is(err, tt.want) || body.closed.Load() != 1 {
				t.Fatalf("err=%v want=%v closes=%d", err, tt.want, body.closed.Load())
			}
		})
	}
}

func TestTransformCancellationAndCloseError(t *testing.T) {
	for _, cancelNow := range []bool{false, true} {
		reader, writer := io.Pipe()
		ctx, cancel := context.WithCancel(context.Background())
		if cancelNow {
			cancel()
		}
		start := time.Now()
		data, err := (Mapping{}).transform(ctx, reader, "store", nil, 20*time.Millisecond)
		cancel()
		writer.Close()
		want := context.DeadlineExceeded
		if cancelNow {
			want = context.Canceled
		}
		if !errors.Is(err, want) || data != nil || time.Since(start) > time.Second {
			t.Fatalf("data=%q err=%v duration=%v", data, err, time.Since(start))
		}
	}
	closeErr := errors.New("close failed")
	body := &trackedBody{Reader: strings.NewReader(validXML), err: closeErr}
	data, err := (Mapping{VisibleBucket: "visible", Prefix: "tenant/"}).Transform(context.Background(), body, "store", nil)
	if !errors.Is(err, closeErr) || data != nil || body.closed.Load() != 1 {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestEncodedKeysAndOpaqueTokens(t *testing.T) {
	q := url.Values{"prefix": {"a%2F+"}, "start-after": {"a%2F+雪"}, "encoding-type": {"url"}, "continuation-token": {"%2F+/tenant/&opaque"}}
	m := Mapping{VisibleBucket: "visible", Prefix: "tenant/"}
	mapped, err := m.Query(q)
	if err != nil || mapped.Get("prefix") != "tenant/a%2F+" || mapped.Get("start-after") != "tenant/a%2F+雪" || mapped.Get("continuation-token") != q.Get("continuation-token") || q.Get("prefix") != "a%2F+" {
		t.Fatalf("query=%v err=%v", mapped, err)
	}
	input := `<ListBucketResult><Name>store</Name><Prefix>tenant%2Fa%252F%2B</Prefix><StartAfter>tenant%2Fa%252F%2B%E9%9B%AA</StartAfter><EncodingType>url</EncodingType><IsTruncated>true</IsTruncated><ContinuationToken>%2F+/tenant/&amp;opaque</ContinuationToken><NextContinuationToken>%25+next/tenant/&amp;opaque</NextContinuationToken><Contents><Key>tenant%2Fa%252F%2B雪</Key></Contents><CommonPrefixes><Prefix>tenant%2Fa%252F%2Bdir%2F</Prefix></CommonPrefixes></ListBucketResult>`
	data, err := m.Transform(context.Background(), io.NopCloser(strings.NewReader(input)), "store", q)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<Name>visible</Name>", "<Prefix>a%252F%2B</Prefix>", "<StartAfter>a%252F%2B%E9%9B%AA</StartAfter>", "<Key>a%252F%2B%E9%9B%AA</Key>", "<Prefix>a%252F%2Bdir%2F</Prefix>", "<ContinuationToken>%2F+/tenant/&amp;opaque</ContinuationToken>", "<NextContinuationToken>%25+next/tenant/&amp;opaque</NextContinuationToken>"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %q in %s", want, data)
		}
	}
	bad := strings.Replace(input, "tenant%2Fa%252F%2B雪", "%GG", 1)
	if _, err := m.Transform(context.Background(), io.NopCloser(strings.NewReader(bad)), "store", q); !errors.Is(err, ErrResponse) {
		t.Fatal(err)
	}
}

func TestURLEncodedXMLSpaceAndPlusAreDistinct(t *testing.T) {
	for _, space := range []string{"+", "%20"} {
		input := `<ListBucketResult><Name>store</Name><Prefix>tenant` + space + `%2F</Prefix><EncodingType>url</EncodingType><IsTruncated>false</IsTruncated><Contents><Key>tenant` + space + `%2Fa` + space + `%2Bb</Key></Contents></ListBucketResult>`
		data, err := (Mapping{VisibleBucket: "visible", Prefix: "tenant /"}).Transform(context.Background(), io.NopCloser(strings.NewReader(input)), "store", url.Values{"encoding-type": {"url"}})
		if err != nil || !strings.Contains(string(data), "<Key>a%20%2Bb</Key>") {
			t.Fatalf("space encoding=%q body=%s err=%v", space, data, err)
		}
	}
}

func TestEncodedListingAcceptsExactUnencodedQueryEchoes(t *testing.T) {
	input := `<ListBucketResult><Name>store</Name><Prefix>tenant %2F/dir+</Prefix><StartAfter>tenant %2F/dir+a</StartAfter><Delimiter>+</Delimiter><EncodingType>url</EncodingType><IsTruncated>false</IsTruncated><Contents><Key>tenant%20%252F/dir%2Bb</Key></Contents></ListBucketResult>`
	q := url.Values{"prefix": {"dir+"}, "start-after": {"dir+a"}, "delimiter": {"+"}, "encoding-type": {"url"}}
	data, err := (Mapping{VisibleBucket: "visible", Prefix: "tenant %2F/"}).Transform(context.Background(), io.NopCloser(strings.NewReader(input)), "store", q)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<Prefix>dir%2B</Prefix>", "<StartAfter>dir%2Ba</StartAfter>", "<Delimiter>%2B</Delimiter>", "<Key>dir%2Bb</Key>"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %s in %s", want, data)
		}
	}
}
