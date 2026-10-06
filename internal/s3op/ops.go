package s3op

import (
	"fmt"
	"net/http"
	"strings"
)

type Operation string

const (
	GetObject     Operation = "GetObject"
	HeadObject    Operation = "HeadObject"
	PutObject     Operation = "PutObject"
	DeleteObject  Operation = "DeleteObject"
	HeadBucket    Operation = "HeadBucket"
	ListObjectsV2 Operation = "ListObjectsV2"
	ListObjectsV1 Operation = "ListObjectsV1"
	ListBuckets   Operation = "ListBuckets"
	CopyObject    Operation = "CopyObject"
	Unknown       Operation = "Unknown"
)

func IsRead(op Operation) bool {
	switch op {
	case GetObject, HeadObject, ListObjectsV2, ListObjectsV1, ListBuckets, HeadBucket:
		return true
	}
	return false
}

func IsWrite(op Operation) bool {
	switch op {
	case PutObject, DeleteObject, CopyObject:
		return true
	}
	return false
}

func SupportsFanout(op Operation) bool {
	switch op {
	case PutObject, DeleteObject:
		return true
	}
	return false
}

func DeclaredOperations() []Operation {
	return []Operation{GetObject, HeadObject, PutObject, DeleteObject, HeadBucket, ListObjectsV2, ListObjectsV1, ListBuckets, CopyObject, Unknown}
}

func IsConfigurable(op string) bool {
	switch Operation(op) {
	case GetObject, HeadObject, PutObject, DeleteObject, HeadBucket, ListObjectsV2, ListBuckets:
		return true
	}
	return false
}

func ConfigurableOperations() []Operation {
	return []Operation{GetObject, HeadObject, PutObject, DeleteObject, HeadBucket, ListObjectsV2, ListBuckets}
}

func ValidateOutboundShape(op Operation, method, bucket, key string) error {
	var wantMethod string
	var object bool
	switch op {
	case GetObject:
		wantMethod, object = http.MethodGet, true
	case HeadObject:
		wantMethod, object = http.MethodHead, true
	case PutObject:
		wantMethod, object = http.MethodPut, true
	case DeleteObject:
		wantMethod, object = http.MethodDelete, true
	case HeadBucket:
		wantMethod = http.MethodHead
	case ListObjectsV2:
		wantMethod = http.MethodGet
	default:
		return fmt.Errorf("operation is not supported for outbound execution")
	}
	if method != wantMethod {
		return fmt.Errorf("method does not match operation")
	}
	if bucket == "" || strings.ContainsAny(bucket, "/\\%?#") {
		return fmt.Errorf("operation requires a nonempty bucket without path or URL delimiters")
	}
	if object && key == "" {
		return fmt.Errorf("object operation requires a nonempty key")
	}
	if !object && key != "" {
		return fmt.Errorf("bucket operation requires an empty key")
	}
	return nil
}
