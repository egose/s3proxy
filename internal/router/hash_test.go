package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/s3ops"
)

func TestHashIndexFixedVectors(t *testing.T) {
	counts := [...]int{2, 3, 5, 7, 257, 1<<31 - 1}
	tests := []struct {
		name   string
		bucket string
		key    string
		want   [6]int
	}{
		{
			name: "object_low_bit", bucket: "bucket", key: "key1",
			want: [6]int{0, 1, 2, 5, 80, 644475382},
		},
		{
			name: "object_high_bit", bucket: "bucket", key: "key3",
			want: [6]int{1, 0, 2, 5, 32, 1249651610},
		},
		{
			name: "list_low_bit", bucket: "bucket",
			want: [6]int{0, 1, 0, 5, 98, 1426940170},
		},
		{
			name: "list_high_bit", bucket: "tenant-2",
			want: [6]int{0, 1, 1, 5, 128, 1999132839},
		},
		{
			name: "escaped_low_bit", bucket: "bucket", key: "a%2Fb",
			want: [6]int{0, 0, 2, 2, 189, 1181205342},
		},
		{
			name: "escaped_high_bit", bucket: "bucket", key: "a%252Fb",
			want: [6]int{0, 0, 4, 3, 223, 2143751717},
		},
	}
	for _, tt := range tests {
		for i, n := range counts {
			t.Run(fmt.Sprintf("%s/%d", tt.name, n), func(t *testing.T) {
				if got := hashIndex(tt.bucket, tt.key, n); got != tt.want[i] {
					t.Fatalf("hashIndex(%q, %q, %d) = %d, want %d", tt.bucket, tt.key, n, got, tt.want[i])
				}
			})
		}
	}
}

func TestHashIndexWideDestinationCounts(t *testing.T) {
	tests := []struct {
		n    uint64
		want uint64
	}{
		{n: 1 << 31, want: 1249651609},
		{n: 1 << 32, want: 3397135257},
		{n: 1<<63 - 1, want: 3397135257},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.n), func(t *testing.T) {
			if tt.n > uint64(^uint(0)>>1) {
				t.Skip("destination count exceeds native int range")
			}
			got := hashIndex("bucket", "key3", int(tt.n))
			if got < 0 || uint64(got) != tt.want {
				t.Fatalf("hashIndex(bucket, key3, %d) = %d, want %d", tt.n, got, tt.want)
			}
		})
	}
}

func TestHashSelectionGuards(t *testing.T) {
	for _, n := range []int{-int(^uint(0)>>1) - 1, -1, 0, 1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			if got := hashIndex("bucket", "key3", n); got != 0 {
				t.Fatalf("hashIndex with %d destinations = %d, want 0", n, got)
			}
		})
	}
	route := config.Route{ReadPreference: config.ReadHash}
	if got := selectEffectiveRead(route, nil, nil); got != "" {
		t.Fatalf("read with no destinations = %q, want empty", got)
	}
	if got := selectEffectiveRead(route, []string{"replica"}, nil); got != "replica" {
		t.Fatalf("read with one destination = %q, want replica", got)
	}
}

func TestResolve_HashReadSelection(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		bucket    string
		key       string
		query     string
		op        s3ops.Operation
		wantIndex int
	}{
		{name: "object_low_bit", method: http.MethodGet, bucket: "bucket", key: "key1", op: s3ops.OpGetObject, wantIndex: 1},
		{name: "object_high_bit", method: http.MethodGet, bucket: "bucket", key: "key3", op: s3ops.OpGetObject, wantIndex: 0},
		{name: "head_object", method: http.MethodHead, bucket: "bucket", key: "key3", op: s3ops.OpHeadObject, wantIndex: 0},
		{name: "escaped_separator", method: http.MethodGet, bucket: "bucket", key: "a%2Fb", op: s3ops.OpGetObject, wantIndex: 0},
		{name: "lowercase_escape", method: http.MethodGet, bucket: "bucket", key: "a%2fb", op: s3ops.OpGetObject, wantIndex: 2},
		{name: "literal_separator", method: http.MethodGet, bucket: "bucket", key: "a/b", op: s3ops.OpGetObject, wantIndex: 2},
		{name: "escaped_percent", method: http.MethodGet, bucket: "bucket", key: "a%252Fb", op: s3ops.OpGetObject, wantIndex: 0},
		{name: "unicode_percent_slashes", method: http.MethodGet, bucket: "bucket", key: "%E9%9B%AA/%25//a%2Fb", op: s3ops.OpGetObject, wantIndex: 1},
		{name: "leading_repeated_slashes", method: http.MethodGet, bucket: "bucket", key: "/a//b", op: s3ops.OpGetObject, wantIndex: 2},
		{name: "list_low_bit", method: http.MethodGet, bucket: "bucket", query: "list-type=2", op: s3ops.OpListObjectsV2, wantIndex: 1},
		{name: "list_high_bit", method: http.MethodGet, bucket: "tenant-2", query: "list-type=2", op: s3ops.OpListObjectsV2, wantIndex: 1},
		{name: "list_filters", method: http.MethodGet, bucket: "tenant-2", query: "list-type=2&prefix=a%2Fb&start-after=a%2Fb%25", op: s3ops.OpListObjectsV2, wantIndex: 1},
		{name: "list_continuation", method: http.MethodGet, bucket: "tenant-2", query: "list-type=2&prefix=a%2Fb&continuation-token=opaque%2B%2F%3D", op: s3ops.OpListObjectsV2, wantIndex: 1},
		{name: "head_bucket", method: http.MethodHead, bucket: "tenant-2", op: s3ops.OpHeadBucket, wantIndex: 1},
	}
	orders := []struct {
		name         string
		destinations []string
	}{
		{name: "configured_order", destinations: []string{"replica-c", "primary", "replica-a"}},
		{name: "rotated_order", destinations: []string{"primary", "replica-a", "replica-c"}},
	}
	for _, tt := range tests {
		for _, order := range orders {
			for _, addressing := range []string{"path_style", "virtual_hosted"} {
				t.Run(tt.name+"/"+order.name+"/"+addressing, func(t *testing.T) {
					url := "http://proxy.example/" + tt.bucket + "/" + tt.key
					if addressing == "virtual_hosted" {
						url = "http://" + tt.bucket + ".proxy.example/" + tt.key
					}
					if tt.query != "" {
						url += "?" + tt.query
					}
					ctx, err := requestctx.FromRequest(httptest.NewRequest(tt.method, url, nil), config.Addressing{
						PathStyle: true, VirtualHosted: true, HostSuffixes: []string{"proxy.example"},
					})
					if err != nil {
						t.Fatal(err)
					}
					if ctx.Bucket != tt.bucket || ctx.Key != tt.key {
						t.Fatalf("inbound bucket/key = %q/%q, want %q/%q", ctx.Bucket, ctx.Key, tt.bucket, tt.key)
					}
					if op, err := s3ops.Classify(ctx); err != nil || op != tt.op {
						t.Fatalf("Classify = %q, %v, want %q", op, err, tt.op)
					}
					route := config.Route{
						Name: "hashed", ParserRef: "bucket",
						Operations:      []string{string(tt.op)},
						DestinationRefs: order.destinations,
						Dispatch:        config.DispatchFirst,
						OnMatch:         config.MatchStop,
						ReadPreference:  config.ReadHash,
						Rewrite:         config.RewriteRule{Bucket: "backend", PrependKeyPrefix: "archive/"},
					}
					r := NewResolver([]config.Route{route}, map[string]config.Parser{
						"bucket": {Name: "bucket", Kind: config.ParserBucketExact, Bucket: tt.bucket},
					}, []string{"replica-a", "replica-c", "primary"})
					matches, err := r.Resolve(ctx, tt.op)
					if err != nil {
						t.Fatal(err)
					}
					if len(matches) != 1 {
						t.Fatalf("matches = %d, want 1", len(matches))
					}
					match := matches[0]
					if !slices.Equal(match.Destinations, order.destinations) {
						t.Fatalf("destinations = %v, want %v", match.Destinations, order.destinations)
					}
					if want := order.destinations[tt.wantIndex]; match.EffectiveRead != want {
						t.Errorf("EffectiveRead = %q, want %q (index %d)", match.EffectiveRead, want, tt.wantIndex)
					}
					result, err := rewrite.New().Apply(ctx, match.Route.Rewrite, match.Captures)
					if err != nil {
						t.Fatal(err)
					}
					wantKey := ""
					if tt.key != "" {
						wantKey = "archive/" + tt.key
					}
					if result.Bucket != "backend" || result.Key != wantKey {
						t.Fatalf("rewritten bucket/key = %q/%q, want backend/%q", result.Bucket, result.Key, wantKey)
					}
					if ctx.Bucket != tt.bucket || ctx.Key != tt.key {
						t.Fatal("rewrite changed original hash inputs")
					}
				})
			}
		}
	}
}

func TestResolve_HashWritesUseFirstDestination(t *testing.T) {
	destinations := []string{"replica-c", "primary", "replica-a"}
	for _, dispatch := range []config.DispatchMode{config.DispatchFirst, config.DispatchAll} {
		route := config.Route{
			Name: "mixed", ParserRef: "bucket",
			Operations:      []string{"GetObject", "PutObject", "DeleteObject"},
			DestinationRefs: destinations,
			Dispatch:        dispatch,
			OnMatch:         config.MatchStop,
			ReadPreference:  config.ReadHash,
		}
		r := NewResolver([]config.Route{route}, map[string]config.Parser{
			"bucket": {Name: "bucket", Kind: config.ParserBucketExact, Bucket: "bucket"},
		}, []string{"primary", "replica-a", "replica-c"})
		for _, op := range []s3ops.Operation{s3ops.OpPutObject, s3ops.OpDeleteObject} {
			t.Run(string(dispatch)+"/"+string(op), func(t *testing.T) {
				matches, err := r.Resolve(&requestctx.Context{Bucket: "bucket", Key: "key1"}, op)
				if err != nil {
					t.Fatal(err)
				}
				if len(matches) != 1 {
					t.Fatalf("matches = %d, want 1", len(matches))
				}
				match := matches[0]
				if match.EffectiveRead != "replica-c" {
					t.Fatalf("write target = %q, want replica-c", match.EffectiveRead)
				}
				if !slices.Equal(match.Destinations, destinations) || match.Route.Dispatch != dispatch {
					t.Fatalf("write destinations/dispatch = %v/%s, want %v/%s", match.Destinations, match.Route.Dispatch, destinations, dispatch)
				}
			})
		}
	}
}

func TestResolve_HashDestinationGuards(t *testing.T) {
	route := config.Route{
		Name: "hashed", ParserRef: "bucket",
		Operations:      []string{"GetObject"},
		DestinationRefs: []string{"missing", "replica"},
		Dispatch:        config.DispatchFirst,
		OnMatch:         config.MatchStop,
		ReadPreference:  config.ReadHash,
	}
	parsers := map[string]config.Parser{
		"bucket": {Name: "bucket", Kind: config.ParserBucketExact, Bucket: "bucket"},
	}
	ctx := &requestctx.Context{Bucket: "bucket", Key: "key3"}
	for _, targets := range [][]string{nil, {"replica"}} {
		t.Run(fmt.Sprint(len(targets)), func(t *testing.T) {
			r := NewResolver([]config.Route{route}, parsers, targets)
			matches, err := r.Resolve(ctx, s3ops.OpGetObject)
			if len(targets) == 0 {
				if err == nil || len(matches) != 0 {
					t.Fatalf("no-target Resolve = %v, %v, want no match and an error", matches, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(matches) != 1 || matches[0].EffectiveRead != "replica" || !slices.Equal(matches[0].Destinations, targets) {
				t.Fatalf("one-target Resolve = %v, want only replica", matches)
			}
		})
	}
}
