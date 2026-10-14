package listbuckets

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/config"
)

func BenchmarkList(b *testing.B) {
	for _, size := range []int{8, 4096} {
		b.Run(fmt.Sprintf("Buckets%d", size), func(b *testing.B) {
			buckets := make([]config.VirtualBucket, size)
			dense := make([]string, size)
			var sparse []string
			for i := range buckets {
				name := fmt.Sprintf("bucket-%06d", i)
				buckets[i] = config.VirtualBucket{Name: name, VisibleName: name}
				dense[i] = name
				if i%100 == 0 {
					sparse = append(sparse, name)
				}
			}
			rand.New(rand.NewSource(1)).Shuffle(len(buckets), func(i, j int) {
				buckets[i], buckets[j] = buckets[j], buckets[i]
			})
			svc := New(buckets, time.Unix(1700000000, 0))
			cases := []struct {
				name      string
				principal *auth.Principal
				want      int
			}{
				{name: "Nil", want: size},
				{name: "Wildcard", principal: &auth.Principal{VisibleBuckets: []string{"*"}}, want: size},
				{name: "Sparse", principal: &auth.Principal{VisibleBuckets: sparse}, want: len(sparse)},
				{name: "Dense", principal: &auth.Principal{VisibleBuckets: dense}, want: size},
			}
			for _, tc := range cases {
				b.Run(tc.name, func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						views := svc.List(tc.principal)
						if len(views) != tc.want {
							b.Fatalf("len(views) = %d, want %d", len(views), tc.want)
						}
					}
				})
			}
		})
	}
}
