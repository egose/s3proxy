package s3ops

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/egose/s3proxy/internal/requestctx"
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

func TestClassify_RootUnsupportedMethods(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			ctx := &requestctx.Context{Method: method}
			op, _ := Classify(ctx)
			if op != OpUnknown {
				t.Errorf("expected Unknown, got %s", op)
			}
		})
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

func TestClassify_PutBucketUnsupported(t *testing.T) {
	ctx := &requestctx.Context{
		Method:  http.MethodPut,
		Bucket:  "mybucket",
		Key:     "",
		Headers: http.Header{},
	}
	op, _ := Classify(ctx)
	if op != OpUnknown {
		t.Errorf("expected Unknown, got %s", op)
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

func TestClassify_DeleteBucketUnsupported(t *testing.T) {
	ctx := &requestctx.Context{
		Method: http.MethodDelete,
		Bucket: "mybucket",
		Key:    "",
	}
	op, _ := Classify(ctx)
	if op != OpUnknown {
		t.Errorf("expected Unknown, got %s", op)
	}
}

func TestClassify_SupportedQueries(t *testing.T) {
	tests := []struct {
		name   string
		ctx    requestctx.Context
		wantOp Operation
	}{
		{
			name: "presigned get object",
			ctx: requestctx.Context{
				Method: http.MethodGet,
				Bucket: "mybucket",
				Key:    "file.txt",
				Query: url.Values{
					"X-Amz-Algorithm":     []string{"AWS4-HMAC-SHA256"},
					"X-Amz-Credential":    []string{"ak/20260823/us-east-1/s3/aws4_request"},
					"X-Amz-Date":          []string{"20260823T000000Z"},
					"X-Amz-Expires":       []string{"300"},
					"X-Amz-SignedHeaders": []string{"host"},
					"X-Amz-Signature":     []string{"abcdef"},
				},
			},
			wantOp: OpGetObject,
		},
		{
			name: "aws sdk x-id get object",
			ctx: requestctx.Context{
				Method: http.MethodGet,
				Bucket: "mybucket",
				Key:    "file.txt",
				Query:  url.Values{"x-id": []string{"GetObject"}},
			},
			wantOp: OpGetObject,
		},
		{
			name: "list objects v2",
			ctx: requestctx.Context{
				Method: http.MethodGet,
				Bucket: "mybucket",
				Query: url.Values{
					"list-type":          []string{"2"},
					"prefix":             []string{"logs/"},
					"delimiter":          []string{"/"},
					"continuation-token": []string{"token"},
					"start-after":        []string{"logs/2026/"},
					"max-keys":           []string{"10"},
					"fetch-owner":        []string{"true"},
					"encoding-type":      []string{"url"},
					"x-id":               []string{"ListObjectsV2"},
				},
			},
			wantOp: OpListObjectsV2,
		},
		{
			name: "put object x-id",
			ctx: requestctx.Context{
				Method:  http.MethodPut,
				Bucket:  "mybucket",
				Key:     "file.txt",
				Headers: http.Header{},
				Query:   url.Values{"x-id": []string{"PutObject"}},
			},
			wantOp: OpPutObject,
		},
		{
			name: "delete object x-id",
			ctx: requestctx.Context{
				Method: http.MethodDelete,
				Bucket: "mybucket",
				Key:    "file.txt",
				Query:  url.Values{"x-id": []string{"DeleteObject"}},
			},
			wantOp: OpDeleteObject,
		},
		{
			name: "head object x-id",
			ctx: requestctx.Context{
				Method:  http.MethodHead,
				Bucket:  "mybucket",
				Key:     "file.txt",
				Headers: http.Header{},
				Query:   url.Values{"x-id": []string{"HeadObject"}},
			},
			wantOp: OpHeadObject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, _ := Classify(&tt.ctx)
			if op != tt.wantOp {
				t.Fatalf("operation = %s, want %s", op, tt.wantOp)
			}
		})
	}
}

func TestClassify_UnsupportedQueryOperations(t *testing.T) {
	tests := []struct {
		name   string
		method string
		bucket string
		key    string
		query  string
	}{
		{name: "get acl", method: http.MethodGet, bucket: "mybucket", key: "file.txt", query: "acl="},
		{name: "put acl", method: http.MethodPut, bucket: "mybucket", key: "file.txt", query: "acl="},
		{name: "get tagging", method: http.MethodGet, bucket: "mybucket", key: "file.txt", query: "tagging="},
		{name: "put tagging", method: http.MethodPut, bucket: "mybucket", key: "file.txt", query: "tagging="},
		{name: "delete tagging", method: http.MethodDelete, bucket: "mybucket", key: "file.txt", query: "tagging="},
		{name: "retention", method: http.MethodGet, bucket: "mybucket", key: "file.txt", query: "retention="},
		{name: "legal hold", method: http.MethodPut, bucket: "mybucket", key: "file.txt", query: "legal-hold="},
		{name: "torrent", method: http.MethodGet, bucket: "mybucket", key: "file.txt", query: "torrent="},
		{name: "bucket versioning", method: http.MethodGet, bucket: "mybucket", query: "versioning="},
		{name: "object version id", method: http.MethodGet, bucket: "mybucket", key: "file.txt", query: "versionId=1"},
		{name: "object versions", method: http.MethodGet, bucket: "mybucket", query: "versions="},
		{name: "restore", method: http.MethodPost, bucket: "mybucket", key: "file.txt", query: "restore="},
		{name: "select", method: http.MethodPost, bucket: "mybucket", key: "file.txt", query: "select=&select-type=2"},
		{name: "response override", method: http.MethodGet, bucket: "mybucket", key: "file.txt", query: "response-content-type=text%2Fplain"},
		{name: "multipart create", method: http.MethodPost, bucket: "mybucket", key: "file.txt", query: "uploads="},
		{name: "multipart upload part", method: http.MethodPut, bucket: "mybucket", key: "file.txt", query: "partNumber=1&uploadId=abc"},
		{name: "multipart complete", method: http.MethodPost, bucket: "mybucket", key: "file.txt", query: "uploadId=abc"},
		{name: "list v2 with unsupported subresource", method: http.MethodGet, bucket: "mybucket", query: "list-type=2&acl="},
		{name: "unrecognized query", method: http.MethodGet, bucket: "mybucket", key: "file.txt", query: "foo=bar"},
		{name: "wrong x-id", method: http.MethodGet, bucket: "mybucket", key: "file.txt", query: "x-id=PutObject"},
		{name: "duplicate list type", method: http.MethodGet, bucket: "mybucket", query: "list-type=2&list-type=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatal(err)
			}
			op, _ := Classify(&requestctx.Context{
				Method:  tt.method,
				Bucket:  tt.bucket,
				Key:     tt.key,
				Query:   query,
				Headers: http.Header{},
			})
			if op != OpUnknown {
				t.Fatalf("operation = %s, want %s", op, OpUnknown)
			}
		})
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

func TestConfigurableOperationsMatchRuntimeCapabilities(t *testing.T) {
	tests := []struct {
		op           Operation
		configurable bool
		read         bool
		write        bool
		fanout       bool
	}{
		{OpGetObject, true, true, false, false},
		{OpHeadObject, true, true, false, false},
		{OpPutObject, true, false, true, true},
		{OpDeleteObject, true, false, true, true},
		{OpHeadBucket, true, true, false, false},
		{OpListObjectsV2, true, true, false, false},
		{OpListBuckets, true, true, false, false},
		{OpListObjectsV1, false, true, false, false},
		{OpCopyObject, false, false, true, false},
		{OpUnknown, false, false, false, false},
	}
	configured := map[Operation]bool{}
	for _, op := range ConfigurableOperations() {
		configured[op] = true
	}
	for _, tt := range tests {
		t.Run(string(tt.op), func(t *testing.T) {
			if got := IsConfigurable(string(tt.op)); got != tt.configurable {
				t.Fatalf("IsConfigurable = %v, want %v", got, tt.configurable)
			}
			if got := configured[tt.op]; got != tt.configurable {
				t.Fatalf("ConfigurableOperations contains op = %v, want %v", got, tt.configurable)
			}
			if got := IsRead(tt.op); got != tt.read {
				t.Fatalf("IsRead = %v, want %v", got, tt.read)
			}
			if got := IsWrite(tt.op); got != tt.write {
				t.Fatalf("IsWrite = %v, want %v", got, tt.write)
			}
			if got := SupportsFanout(tt.op); got != tt.fanout {
				t.Fatalf("SupportsFanout = %v, want %v", got, tt.fanout)
			}
		})
	}
}
