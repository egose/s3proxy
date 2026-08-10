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

func Classify(ctx *requestctx.Context) (Operation, error) {
	if ctx.Bucket == "" {
		if ctx.Method != http.MethodGet {
			return OpUnknown, nil
		}
		return OpListBuckets, nil
	}

	if ctx.Method == http.MethodHead && ctx.Key == "" {
		return OpHeadBucket, nil
	}

	switch ctx.Method {
	case http.MethodGet:
		if ctx.Key == "" {
			if ctx.Query.Get("list-type") == "2" {
				return OpListObjectsV2, nil
			}
			return OpListObjectsV1, nil
		}
		return OpGetObject, nil
	case http.MethodHead:
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
		return OpPutObject, nil
	case http.MethodDelete:
		if ctx.Key == "" {
			return OpUnknown, nil
		}
		return OpDeleteObject, nil
	}

	return OpUnknown, nil
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
