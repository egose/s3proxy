package listbuckets

import (
	"reflect"
	"slices"
	"sync"
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
	startup := time.Unix(1700000000, 0)
	buckets := []config.VirtualBucket{
		{Name: "a", VisibleName: "zebra"},
		{Name: "b", VisibleName: "alpha"},
		{Name: "c", VisibleName: "middle"},
	}
	original := slices.Clone(buckets)
	svc := New(buckets, startup)
	if !slices.Equal(buckets, original) {
		t.Fatalf("New mutated its input: got %v, want %v", buckets, original)
	}
	for i := range buckets {
		buckets[i].VisibleName = "mutated"
	}
	buckets = append(buckets, config.VirtualBucket{Name: "d", VisibleName: "logs"})

	want := []BucketView{
		{Name: "alpha", CreationDate: startup},
		{Name: "middle", CreationDate: startup},
		{Name: "zebra", CreationDate: startup},
	}
	if got := svc.List(nil); !slices.Equal(got, want) {
		t.Fatalf("List after input mutation = %v, want %v", got, want)
	}
}

func TestList_VisibilityAndStartupDate(t *testing.T) {
	startup := time.Date(2024, time.March, 4, 5, 6, 7, 123456789, time.FixedZone("startup", 3600))
	svc := New([]config.VirtualBucket{
		{Name: "catalog.zebra", VisibleName: "zebra", RouteRef: "route.zebra"},
		{Name: "catalog.alpha", VisibleName: "alpha", RouteRef: "route.alpha"},
		{Name: "catalog.middle", VisibleName: "middle", RouteRef: "route.middle"},
	}, startup)
	all := []BucketView{
		{Name: "alpha", CreationDate: startup},
		{Name: "middle", CreationDate: startup},
		{Name: "zebra", CreationDate: startup},
	}
	cases := []struct {
		name      string
		principal *auth.Principal
		want      []BucketView
	}{
		{name: "nil principal", want: all},
		{name: "wildcard", principal: &auth.Principal{VisibleBuckets: []string{"*"}}, want: all},
		{name: "wildcard among names", principal: &auth.Principal{VisibleBuckets: []string{"unknown", "zebra", "*", "alpha"}}, want: all},
		{name: "nil policy", principal: &auth.Principal{}},
		{name: "empty policy", principal: &auth.Principal{VisibleBuckets: []string{}}},
		{name: "explicit subset", principal: &auth.Principal{VisibleBuckets: []string{"zebra", "alpha"}}, want: []BucketView{all[0], all[2]}},
		{name: "explicit all", principal: &auth.Principal{VisibleBuckets: []string{"zebra", "middle", "alpha"}}, want: all},
		{name: "duplicates and unknown names", principal: &auth.Principal{VisibleBuckets: []string{"unknown", "middle", "middle", "alpha"}}, want: all[:2]},
		{name: "exact visible names only", principal: &auth.Principal{VisibleBuckets: []string{"ALPHA", "alph", "al*", "catalog.alpha"}}},
		{name: "ungranted route stays visible", principal: &auth.Principal{AllowRoutes: []string{"route.zebra"}, VisibleBuckets: []string{"alpha"}}, want: all[:1]},
		{name: "route grants do not add visibility", principal: &auth.Principal{AllowRoutes: []string{"*"}, VisibleBuckets: []string{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := svc.List(tc.principal); !slices.Equal(got, tc.want) {
				t.Fatalf("List = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestList_EmptyCatalog(t *testing.T) {
	for _, buckets := range [][]config.VirtualBucket{nil, {}} {
		svc := New(buckets, time.Unix(1700000000, 0))
		for _, principal := range []*auth.Principal{
			nil,
			{},
			{VisibleBuckets: []string{}},
			{VisibleBuckets: []string{"*"}},
			{VisibleBuckets: []string{"alpha"}},
		} {
			if got := svc.List(principal); len(got) != 0 {
				t.Fatalf("List on empty catalog = %v, want no buckets", got)
			}
		}
	}
}

func TestList_PrincipalOwnership(t *testing.T) {
	startup := time.Unix(1700000000, 0)
	svc := New([]config.VirtualBucket{
		{VisibleName: "zebra"},
		{VisibleName: "alpha"},
		{VisibleName: "middle"},
	}, startup)
	principal := &auth.Principal{
		Name:           "client",
		AccessKey:      "client-key",
		AllowRoutes:    []string{"route.zebra", "route.alpha"},
		AllowOps:       []string{"GetObject", "ListBuckets"},
		VisibleBuckets: []string{"zebra", "alpha", "zebra", "unknown"},
	}
	original := *principal
	original.AllowRoutes = slices.Clone(principal.AllowRoutes)
	original.AllowOps = slices.Clone(principal.AllowOps)
	original.VisibleBuckets = slices.Clone(principal.VisibleBuckets)
	first := svc.List(principal)
	wantFirst := []BucketView{{Name: "alpha", CreationDate: startup}, {Name: "zebra", CreationDate: startup}}
	if !slices.Equal(first, wantFirst) {
		t.Fatalf("List = %v, want %v", first, wantFirst)
	}
	if !reflect.DeepEqual(*principal, original) {
		t.Fatal("List mutated its principal")
	}

	for i := range principal.VisibleBuckets {
		principal.VisibleBuckets[i] = "middle"
	}
	want := []BucketView{{Name: "middle", CreationDate: startup}}
	if got := svc.List(principal); !slices.Equal(got, want) {
		t.Fatalf("List after policy mutation = %v, want %v", got, want)
	}
	other := &auth.Principal{Name: principal.Name, AccessKey: principal.AccessKey, VisibleBuckets: []string{"alpha"}}
	want = []BucketView{{Name: "alpha", CreationDate: startup}}
	if got := svc.List(other); !slices.Equal(got, want) {
		t.Fatalf("List for another principal with the same identity = %v, want %v", got, want)
	}
	principal.VisibleBuckets = []string{}
	if got := svc.List(principal); len(got) != 0 {
		t.Fatalf("List after denying visibility = %v, want no buckets", got)
	}
	if !slices.Equal(first, wantFirst) {
		t.Fatalf("prior result changed after policy mutation: got %v, want %v", first, wantFirst)
	}
}

func TestList_ResultMutationIsolation(t *testing.T) {
	startup := time.Unix(1700000000, 0)
	svc := New([]config.VirtualBucket{
		{VisibleName: "zebra"},
		{VisibleName: "alpha"},
		{VisibleName: "middle"},
	}, startup)
	all := []BucketView{
		{Name: "alpha", CreationDate: startup},
		{Name: "middle", CreationDate: startup},
		{Name: "zebra", CreationDate: startup},
	}
	cases := []struct {
		name      string
		principal *auth.Principal
		want      []BucketView
	}{
		{name: "nil", want: all},
		{name: "wildcard", principal: &auth.Principal{VisibleBuckets: []string{"*"}}, want: all},
		{name: "subset", principal: &auth.Principal{VisibleBuckets: []string{"zebra", "alpha"}}, want: []BucketView{all[0], all[2]}},
		{name: "explicit all", principal: &auth.Principal{VisibleBuckets: []string{"middle", "zebra", "alpha"}}, want: all},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first := svc.List(tc.principal)
			second := svc.List(tc.principal)
			if !slices.Equal(first, tc.want) || !slices.Equal(second, tc.want) {
				t.Fatalf("initial lists = %v, %v, want %v", first, second, tc.want)
			}
			for i := range first {
				first[i] = BucketView{Name: "mutated", CreationDate: startup.Add(time.Hour)}
			}
			first = append(first[:1], BucketView{Name: "appended"})
			if !slices.Equal(second, tc.want) {
				t.Fatalf("mutating one result changed another: got %v, want %v", second, tc.want)
			}
			if got := svc.List(tc.principal); !slices.Equal(got, tc.want) {
				t.Fatalf("List after result mutation = %v, want %v", got, tc.want)
			}
			if got := svc.List(nil); !slices.Equal(got, all) {
				t.Fatalf("full catalog after result mutation = %v, want %v", got, all)
			}
		})
	}
}

func TestList_ConcurrentDiscoveryWithResultMutation(t *testing.T) {
	startup := time.Unix(1700000000, 0)
	svc := New([]config.VirtualBucket{
		{VisibleName: "zebra"},
		{VisibleName: "alpha"},
		{VisibleName: "middle"},
	}, startup)
	all := []BucketView{
		{Name: "alpha", CreationDate: startup},
		{Name: "middle", CreationDate: startup},
		{Name: "zebra", CreationDate: startup},
	}
	cases := []struct {
		principal *auth.Principal
		want      []BucketView
	}{
		{want: all},
		{principal: &auth.Principal{VisibleBuckets: []string{"*"}}, want: all},
		{principal: &auth.Principal{VisibleBuckets: []string{"zebra", "alpha"}}, want: []BucketView{all[0], all[2]}},
		{principal: &auth.Principal{VisibleBuckets: []string{}}},
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 100; j++ {
				for _, tc := range cases {
					views := svc.List(tc.principal)
					if !slices.Equal(views, tc.want) {
						t.Errorf("concurrent List = %v, want %v", views, tc.want)
						return
					}
					for k := range views {
						views[k] = BucketView{Name: "mutated"}
					}
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := svc.List(nil); !slices.Equal(got, all) {
		t.Fatalf("List after concurrent result mutation = %v, want %v", got, all)
	}
}
