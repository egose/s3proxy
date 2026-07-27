package s3ops

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jahn/s3proxy/internal/requestctx"
)

type Operation string

const (
	OpGetObject     Operation = "GetObject"
	OpHeadObject    Operation = "HeadObject"
	OpPutObject     Operation = "PutObject"
	OpDeleteObject  Operation = "DeleteObject"
	OpHeadBucket    Operation = "HeadBucket"
	OpListObjectsV2 Operation = "ListObjectsV2"
	OpListObjectsV1 Operation = "ListObjectsV1"
	OpListBuckets   Operation = "ListBuckets"
	OpCopyObject    Operation = "CopyObject"
	OpUnknown       Operation = "Unknown"
)

func Classify(ctx *requestctx.Context) (Operation, error) {
	if ctx.Bucket == "" {
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
		if ctx.Headers.Get("x-amz-copy-source") != "" {
			return OpCopyObject, nil
		}
		if ctx.Query.Has("uploads") {
			return OpUnknown, nil
		}
		return OpPutObject, nil
	case http.MethodDelete:
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
	switch op {
	case OpGetObject, OpHeadObject, OpListObjectsV2, OpListObjectsV1, OpListBuckets, OpHeadBucket:
		return true
	}
	return false
}

func IsWrite(op Operation) bool {
	switch op {
	case OpPutObject, OpDeleteObject, OpCopyObject:
		return true
	}
	return false
}

func SupportsFanout(op Operation) bool {
	switch op {
	case OpPutObject, OpDeleteObject:
		return true
	}
	return false
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
