package replaybody

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func BenchmarkEnsureReplay(b *testing.B) {
	cases := []struct {
		name  string
		size  int64
		known bool
	}{
		{name: "Known/1KiB", size: 1 << 10, known: true},
		{name: "Known/1MiB", size: 1 << 20, known: true},
		{name: "Known/Max", size: DefaultMaxBytes, known: true},
		{name: "Unknown/1KiB", size: 1 << 10},
		{name: "Unknown/1MiB", size: 1 << 20},
		{name: "Unknown/Max", size: DefaultMaxBytes},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			payload := bytes.Repeat([]byte("x"), int(tc.size))
			b.SetBytes(tc.size)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				budget := NewBudget(DefaultMaxBytes, DefaultAggregateMaxBytes)
				req, err := http.NewRequest(http.MethodPut, "http://proxy.local/bucket/key", io.NopCloser(bytes.NewReader(payload)))
				if err != nil {
					b.Fatal(err)
				}
				if tc.known {
					req.ContentLength = tc.size
				} else {
					req.ContentLength = -1
				}
				if err := budget.Ensure(req); err != nil {
					b.Fatal(err)
				}
				if err := Release(req); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
