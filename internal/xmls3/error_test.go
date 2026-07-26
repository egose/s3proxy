package xmls3

import (
	"encoding/xml"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, 404, "NoSuchBucket", "bucket not found", "req-123")
	if w.Code != 404 {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "NoSuchBucket") {
		t.Errorf("expected XML to contain NoSuchBucket, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "req-123") {
		t.Errorf("expected XML to contain request id, got %s", w.Body.String())
	}
}

func TestWriteNotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	WriteNotImplemented(w, "req-456")
	if w.Code != 501 {
		t.Errorf("expected status 501, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "NotImplemented") {
		t.Errorf("expected XML to contain NotImplemented")
	}
}

func TestWriteListBuckets(t *testing.T) {
	w := httptest.NewRecorder()
	buckets := []BucketEntry{
		{Name: "bucket-a", CreationDate: "2024-01-01T00:00:00Z"},
		{Name: "bucket-b", CreationDate: "2024-01-02T00:00:00Z"},
	}
	WriteListBuckets(w, "owner-id", "owner-name", buckets)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "ListAllMyBucketsResult") {
		t.Errorf("expected ListAllMyBucketsResult in XML")
	}
	if !strings.Contains(body, "bucket-a") {
		t.Errorf("expected bucket-a in XML")
	}
	if !strings.Contains(body, "bucket-b") {
		t.Errorf("expected bucket-b in XML")
	}
	var result listAllMyBucketsResult
	if err := xml.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}
	if len(result.Buckets.Bucket) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(result.Buckets.Bucket))
	}
}
