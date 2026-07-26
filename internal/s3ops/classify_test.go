package s3ops

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/jahn/s3proxy/internal/requestctx"
)

func TestClassify_ListBuckets(t *testing.T) {
	ctx := &requestctx.Context{
		Method: http.MethodGet,
	}
	op, _ := Classify(ctx)
	if op != OpListBuckets {
		t.Errorf("expected ListBuckets, got %s", op)
	}
}

func TestClassify_GetObject(t *testing.T) {
	ctx := &requestctx.Context{
		Method: http.MethodGet,
		Bucket: "mybucket",
		Key:    "file.txt",
	}
	op, _ := Classify(ctx)
	if op != OpGetObject {
		t.Errorf("expected GetObject, got %s", op)
	}
}

func TestClassify_HeadObject(t *testing.T) {
	ctx := &requestctx.Context{
		Method:  http.MethodHead,
		Bucket:  "mybucket",
		Key:     "file.txt",
		Headers: http.Header{},
	}
	op, _ := Classify(ctx)
	if op != OpHeadObject {
		t.Errorf("expected HeadObject, got %s", op)
	}
}

func TestClassify_HeadBucket(t *testing.T) {
	ctx := &requestctx.Context{
		Method:  http.MethodHead,
		Bucket:  "mybucket",
		Key:     "",
		Headers: http.Header{},
	}
	op, _ := Classify(ctx)
	if op != OpHeadBucket {
		t.Errorf("expected HeadBucket, got %s", op)
	}
}

func TestClassify_ListObjectsV2(t *testing.T) {
	ctx := &requestctx.Context{
		Method: http.MethodGet,
		Bucket: "mybucket",
		Query:  url.Values{"list-type": []string{"2"}},
	}
	op, _ := Classify(ctx)
	if op != OpListObjectsV2 {
		t.Errorf("expected ListObjectsV2, got %s", op)
	}
}

func TestClassify_ListObjectsV1(t *testing.T) {
	ctx := &requestctx.Context{
		Method: http.MethodGet,
		Bucket: "mybucket",
		Query:  url.Values{},
	}
	op, _ := Classify(ctx)
	if op != OpListObjectsV1 {
		t.Errorf("expected ListObjectsV1, got %s", op)
	}
}

func TestClassify_PutObject(t *testing.T) {
	ctx := &requestctx.Context{
		Method:  http.MethodPut,
		Bucket:  "mybucket",
		Key:     "file.txt",
		Headers: http.Header{},
	}
	op, _ := Classify(ctx)
	if op != OpPutObject {
		t.Errorf("expected PutObject, got %s", op)
	}
}

func TestClassify_DeleteObject(t *testing.T) {
	ctx := &requestctx.Context{
		Method: http.MethodDelete,
		Bucket: "mybucket",
		Key:    "file.txt",
	}
	op, _ := Classify(ctx)
	if op != OpDeleteObject {
		t.Errorf("expected DeleteObject, got %s", op)
	}
}

func TestIsMultipart_UploadId(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "uploadId=abc123"}}
	if !IsMultipart(r) {
		t.Error("expected multipart detection with uploadId")
	}
}

func TestIsMultipart_Uploads(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "uploads="}}
	if !IsMultipart(r) {
		t.Error("expected multipart detection with uploads")
	}
}

func TestIsMultipart_False(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "list-type=2"}}
	if IsMultipart(r) {
		t.Error("did not expect multipart detection")
	}
}
