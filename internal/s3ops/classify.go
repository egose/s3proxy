package s3ops

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/s3op"
)

type Operation = s3op.Operation

const (
	OpGetObject     = s3op.GetObject
	OpHeadObject    = s3op.HeadObject
	OpPutObject     = s3op.PutObject
	OpDeleteObject  = s3op.DeleteObject
	OpHeadBucket    = s3op.HeadBucket
	OpListObjectsV2 = s3op.ListObjectsV2
	OpListObjectsV1 = s3op.ListObjectsV1
	OpListBuckets   = s3op.ListBuckets
	OpCopyObject    = s3op.CopyObject
	OpUnknown       = s3op.Unknown
)

var (
	baseQueryKeys = map[string]bool{"x-id": true}
	listV2Keys    = map[string]bool{
		"continuation-token": true,
		"delimiter":          true,
		"encoding-type":      true,
		"fetch-owner":        true,
		"list-type":          true,
		"max-keys":           true,
		"prefix":             true,
		"start-after":        true,
		"x-id":               true,
	}
)

func Classify(ctx *requestctx.Context) (Operation, error) {
	if ctx.Bucket == "" {
		if ctx.Method != http.MethodGet {
			return OpUnknown, nil
		}
		if !queryAllowed(ctx.Query, baseQueryKeys) || !xIDAllowed(ctx.Query, OpListBuckets) {
			return OpUnknown, nil
		}
		return OpListBuckets, nil
	}

	if ctx.Method == http.MethodHead && ctx.Key == "" {
		if !queryAllowed(ctx.Query, baseQueryKeys) || !xIDAllowed(ctx.Query, OpHeadBucket) {
			return OpUnknown, nil
		}
		return OpHeadBucket, nil
	}

	switch ctx.Method {
	case http.MethodGet:
		if ctx.Key == "" {
			if exactlyOne(ctx.Query, "list-type", "2") {
				if !queryAllowed(ctx.Query, listV2Keys) || !xIDAllowed(ctx.Query, OpListObjectsV2) {
					return OpUnknown, nil
				}
				return OpListObjectsV2, nil
			}
			if !queryAllowed(ctx.Query, baseQueryKeys) || !xIDAllowed(ctx.Query, OpListObjectsV1) {
				return OpUnknown, nil
			}
			return OpListObjectsV1, nil
		}
		if !queryAllowed(ctx.Query, baseQueryKeys) || !xIDAllowed(ctx.Query, OpGetObject) {
			return OpUnknown, nil
		}
		return OpGetObject, nil
	case http.MethodHead:
		if !queryAllowed(ctx.Query, baseQueryKeys) || !xIDAllowed(ctx.Query, OpHeadObject) {
			return OpUnknown, nil
		}
		return OpHeadObject, nil
	case http.MethodPut:
		if ctx.Key == "" {
			return OpUnknown, nil
		}
		if ctx.Headers.Get("x-amz-copy-source") != "" {
			return OpCopyObject, nil
		}
		if ctx.Query.Has("uploads") {
			return OpUnknown, nil
		}
		if !queryAllowed(ctx.Query, baseQueryKeys) || !xIDAllowed(ctx.Query, OpPutObject) {
			return OpUnknown, nil
		}
		return OpPutObject, nil
	case http.MethodDelete:
		if ctx.Key == "" {
			return OpUnknown, nil
		}
		if !queryAllowed(ctx.Query, baseQueryKeys) || !xIDAllowed(ctx.Query, OpDeleteObject) {
			return OpUnknown, nil
		}
		return OpDeleteObject, nil
	}

	return OpUnknown, nil
}

func queryAllowed(values map[string][]string, allowed map[string]bool) bool {
	for key := range values {
		canonical := strings.ToLower(key)
		if isInboundAuthQueryParam(canonical) {
			continue
		}
		if !allowed[canonical] {
			return false
		}
	}
	return true
}

func xIDAllowed(values map[string][]string, op Operation) bool {
	found := false
	for key, vals := range values {
		if strings.ToLower(key) != "x-id" {
			continue
		}
		if found || len(vals) != 1 || vals[0] != string(op) {
			return false
		}
		found = true
	}
	return true
}

func exactlyOne(values map[string][]string, key, value string) bool {
	vals, ok := values[key]
	return ok && len(vals) == 1 && vals[0] == value
}

func isInboundAuthQueryParam(name string) bool {
	switch strings.ToLower(name) {
	case "x-amz-algorithm", "x-amz-credential", "x-amz-date", "x-amz-expires",
		"x-amz-security-token", "x-amz-signature", "x-amz-signedheaders":
		return true
	default:
		return false
	}
}

func IsMultipart(r *http.Request) bool {
	if r == nil {
		return false
	}
	if _, ok := r.URL.Query()["uploadId"]; ok {
		return true
	}
	if _, ok := r.URL.Query()["uploads"]; ok {
		return true
	}
	if r.Header.Get("x-amz-multipart-upload-id") != "" {
		return true
	}
	return false
}

func IsRead(op Operation) bool {
	return s3op.IsRead(op)
}

func IsWrite(op Operation) bool {
	return s3op.IsWrite(op)
}

func SupportsFanout(op Operation) bool {
	return s3op.SupportsFanout(op)
}

func IsConfigurable(op string) bool {
	return s3op.IsConfigurable(op)
}

func ConfigurableOperations() []Operation {
	return s3op.ConfigurableOperations()
}

func ParseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func ParseKey(key string) []string {
	if key == "" {
		return nil
	}
	return strings.Split(key, "/")
}
