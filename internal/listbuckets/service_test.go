package listbuckets

import (
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/config"
)

func TestList_AllBucketsForNilPrincipal(t *testing.T) {
	svc := New([]config.VirtualBucket{
		{Name: "a", VisibleName: "images"},
		{Name: "b", VisibleName: "logs"},
	}, time.Now())

	views := svc.List(nil)
	if len(views) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(views))
	}
	if views[0].Name != "images" {
		t.Errorf("expected first bucket 'images', got %q", views[0].Name)
	}
}

func TestList_FilteredByPrincipal(t *testing.T) {
	svc := New([]config.VirtualBucket{
		{Name: "a", VisibleName: "images"},
		{Name: "b", VisibleName: "logs"},
		{Name: "c", VisibleName: "secret"},
	}, time.Now())

	p := &auth.Principal{
		Name:           "ci",
		VisibleBuckets: []string{"images", "logs"},
	}

	views := svc.List(p)
	if len(views) != 2 {
		t.Fatalf("expected 2 visible buckets, got %d", len(views))
	}
	for _, v := range views {
		if v.Name == "secret" {
			t.Error("expected 'secret' bucket to be filtered out")
		}
	}
}

func TestList_WildcardPrincipal(t *testing.T) {
	svc := New([]config.VirtualBucket{
		{Name: "a", VisibleName: "images"},
		{Name: "b", VisibleName: "logs"},
	}, time.Now())

	p := &auth.Principal{
		Name:           "admin",
		VisibleBuckets: []string{"*"},
	}

	views := svc.List(p)
	if len(views) != 2 {
		t.Fatalf("expected 2 buckets for wildcard principal, got %d", len(views))
	}
}

func TestList_Sorted(t *testing.T) {
	svc := New([]config.VirtualBucket{
		{Name: "a", VisibleName: "zebra"},
		{Name: "b", VisibleName: "alpha"},
		{Name: "c", VisibleName: "middle"},
	}, time.Now())

	views := svc.List(nil)
	if views[0].Name != "alpha" {
		t.Errorf("expected first sorted bucket 'alpha', got %q", views[0].Name)
	}
	if views[2].Name != "zebra" {
		t.Errorf("expected last sorted bucket 'zebra', got %q", views[2].Name)
	}
}

func TestListSnapshotsBucketsAtConstruction(t *testing.T) {
	buckets := []config.VirtualBucket{
		{Name: "a", VisibleName: "images"},
	}
	svc := New(buckets, time.Now())
	buckets[0].VisibleName = "mutated"
	buckets = append(buckets, config.VirtualBucket{Name: "b", VisibleName: "logs"})

	views := svc.List(nil)
	if got, want := len(views), 1; got != want {
		t.Fatalf("len(views) = %d, want %d", got, want)
	}
	if got, want := views[0].Name, "images"; got != want {
		t.Fatalf("views[0].Name = %q, want %q", got, want)
	}
}
