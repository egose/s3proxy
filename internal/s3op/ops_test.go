package s3op

import (
	"net/http"
	"testing"
)

func TestValidateOutboundShape(t *testing.T) {
	valid := map[Operation]struct{ method, key string }{
		GetObject:     {http.MethodGet, "key"},
		HeadObject:    {http.MethodHead, "key"},
		PutObject:     {http.MethodPut, "key"},
		DeleteObject:  {http.MethodDelete, "key"},
		HeadBucket:    {http.MethodHead, ""},
		ListObjectsV2: {http.MethodGet, ""},
	}
	for _, op := range append(DeclaredOperations(), "", "DeleteBucket") {
		t.Run(string(op), func(t *testing.T) {
			shape, supported := valid[op]
			if err := ValidateOutboundShape(op, shape.method, "bucket", shape.key); (err == nil) != supported {
				t.Fatalf("valid shape error = %v, supported = %t", err, supported)
			}
			if !supported {
				return
			}
			for _, method := range []string{"", "GET", "HEAD", "PUT", "DELETE", "POST"} {
				if err := ValidateOutboundShape(op, method, "bucket", shape.key); (err == nil) != (method == shape.method) {
					t.Errorf("method %q: %v", method, err)
				}
			}
			for _, bucket := range []string{"", "bucket/key", "bucket%2Fkey", "bucket?query", "bucket#fragment", "bucket\\key"} {
				if err := ValidateOutboundShape(op, shape.method, bucket, shape.key); err == nil {
					t.Errorf("accepted bucket %q", bucket)
				}
			}
			for _, key := range []string{"", "/", "//foo", "%2Ffoo", "foo//bar/../"} {
				if err := ValidateOutboundShape(op, shape.method, "bucket", key); (err == nil) != ((key == "") == (shape.key == "")) {
					t.Errorf("key %q: %v", key, err)
				}
			}
		})
	}
}

func TestConfigurableOperationsMatchCapability(t *testing.T) {
	seen := make(map[Operation]bool)
	for _, op := range ConfigurableOperations() {
		if seen[op] {
			t.Fatalf("ConfigurableOperations contains duplicate %q", op)
		}
		seen[op] = true
		if !IsConfigurable(string(op)) {
			t.Fatalf("ConfigurableOperations contains non-configurable operation %q", op)
		}
	}

	for _, op := range DeclaredOperations() {
		if got, want := seen[op], IsConfigurable(string(op)); got != want {
			t.Fatalf("operation %q configurable list membership = %v, want %v", op, got, want)
		}
	}
}

func TestDeclaredOperationCapabilities(t *testing.T) {
	tests := []struct {
		op           Operation
		read         bool
		write        bool
		fanout       bool
		configurable bool
	}{
		{op: GetObject, read: true, configurable: true},
		{op: HeadObject, read: true, configurable: true},
		{op: PutObject, write: true, fanout: true, configurable: true},
		{op: DeleteObject, write: true, fanout: true, configurable: true},
		{op: HeadBucket, read: true, configurable: true},
		{op: ListObjectsV2, read: true, configurable: true},
		{op: ListObjectsV1, read: true},
		{op: ListBuckets, read: true, configurable: true},
		{op: CopyObject, write: true},
		{op: Unknown},
	}

	if got, want := len(tests), len(DeclaredOperations()); got != want {
		t.Fatalf("test table covers %d operations, want %d", got, want)
	}

	seen := make(map[Operation]bool, len(tests))
	for _, tt := range tests {
		tc := tt
		testingName := string(tt.op)
		if testingName == "" {
			testingName = "empty"
		}
		t.Run(testingName, func(t *testing.T) {
			if seen[tc.op] {
				t.Fatalf("duplicate operation %q in capability test table", tc.op)
			}
			seen[tc.op] = true
			if got := IsRead(tc.op); got != tc.read {
				t.Fatalf("IsRead(%q) = %v, want %v", tc.op, got, tc.read)
			}
			if got := IsWrite(tc.op); got != tc.write {
				t.Fatalf("IsWrite(%q) = %v, want %v", tc.op, got, tc.write)
			}
			if got := SupportsFanout(tc.op); got != tc.fanout {
				t.Fatalf("SupportsFanout(%q) = %v, want %v", tc.op, got, tc.fanout)
			}
			if got := IsConfigurable(string(tc.op)); got != tc.configurable {
				t.Fatalf("IsConfigurable(%q) = %v, want %v", tc.op, got, tc.configurable)
			}
		})
	}

	for _, op := range DeclaredOperations() {
		if !seen[op] {
			t.Fatalf("operation %q missing from capability test table", op)
		}
	}
}
