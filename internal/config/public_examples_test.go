package config_test

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/rewrite"
)

func TestPublicConfigExamplesAndBucketSemantics(t *testing.T) {
	blocks := regexp.MustCompile("(?s)```hcl\\n(.*?)```")
	envs := regexp.MustCompile(`env\("([A-Z0-9_]+)"\)`)
	count := 0
	for _, file := range []string{"README.md", "docs/design.md", "website/docs/quickstart.md", "website/docs/configuration.md", "website/docs/config-examples.md", "sandbox/integration-config.hcl"} {
		data, err := os.ReadFile(filepath.Join("../..", file))
		if err != nil {
			t.Fatal(err)
		}
		configs := blocks.FindAllStringSubmatch(string(data), -1)
		if strings.HasSuffix(file, ".hcl") {
			configs = [][]string{{"", string(data)}}
		}
		for _, block := range configs {
			content := block[1]
			if !strings.Contains(content, `listener "http"`) {
				continue
			}
			count++
			for _, env := range envs.FindAllStringSubmatch(content, -1) {
				value := "example-" + env[1]
				if strings.Contains(env[1], "ENDPOINT") {
					value = "http://example.test"
				}
				t.Setenv(env[1], value)
			}
			path := filepath.Join(t.TempDir(), "example.hcl")
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			rt, err := config.LoadFile(path)
			if err != nil {
				t.Fatalf("%s: %v", file, err)
			}
			for _, route := range rt.Routes {
				for _, op := range route.Operations {
					if op != "ListObjectsV2" && op != "HeadBucket" {
						continue
					}
					ctx := &requestctx.Context{Method: "HEAD", Bucket: "visible"}
					if op == "ListObjectsV2" {
						ctx.Method = "GET"
						ctx.Query = url.Values{"list-type": {"2"}}
					}
					rw, err := rewrite.New().Apply(ctx, route.Rewrite, map[string]string{"tenant": "acme"})
					if err != nil || rw.Key != "" {
						t.Fatalf("%s %s %s: result=%+v err=%v", file, route.Name, op, rw, err)
					}
					if op == "ListObjectsV2" {
						if rw.Namespace == nil {
							t.Fatal("missing namespace")
						}
						ctx.Key = "a%252F%20雪//key"
						object, err := rewrite.New().Apply(ctx, route.Rewrite, map[string]string{"tenant": "acme"})
						if err != nil {
							t.Fatal(err)
						}
						got, _ := url.PathUnescape(object.Key)
						key, _ := url.PathUnescape(ctx.Key)
						if got != rw.Namespace.Prefix+key {
							t.Fatalf("%s %s: object/list namespace mismatch", file, route.Name)
						}
					}
				}
			}
		}
	}
	if count < 10 {
		t.Fatalf("only found %d full public configs", count)
	}
}
