package rewrite

import (
	"errors"
	"net/url"
	"testing"
	"text/template"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
)

func TestListingNamespaceMatchesObjectRewrite(t *testing.T) {
	for _, tt := range []struct{ prefix, tmpl, want string }{
		{"", "", ""}, {"assets/", "", "assets/"}, {"/a//", "", "/a//"},
		{"a%252F/", "", "a%2F/"}, {"雪%20%2f/", "", "雪 //"},
		{"a+b/", "", "a+b/"},
		{"", "{{ .Captures.tenant }}/{{ .Key }}", "acme/"},
		{"assets/", "{{ .Bucket }}/{{ .Captures.tenant }}/{{ .Key }}", "store/acme/assets/"},
		{"", "literal{{ .Key }}", "literal"},
	} {
		t.Run(tt.prefix+tt.tmpl, func(t *testing.T) {
			rw := config.RewriteRule{Bucket: "store", PrependKeyPrefix: tt.prefix, KeyTemplate: tt.tmpl}
			if tt.tmpl != "" {
				rw.CompiledTemplate = template.Must(template.New("key").Parse(tt.tmpl))
			}
			ctx := &requestctx.Context{Method: "GET", Bucket: "visible", Query: url.Values{"list-type": {"2"}}}
			result, err := New().Apply(ctx, rw, map[string]string{"tenant": "acme"})
			if err != nil || result.Namespace == nil || result.Key != "" || result.Bucket != "store" {
				t.Fatalf("result = %+v, err = %v", result, err)
			}
			if result.Namespace.Prefix != tt.want || result.Namespace.VisibleBucket != "visible" {
				t.Fatalf("mapping = %+v, want prefix %q", result.Namespace, tt.want)
			}
			for _, key := range []string{"/foo", "//a/../b", "a%252F", "%2F", "雪%20%25"} {
				ctx.Key = key
				object, err := New().Apply(ctx, rw, map[string]string{"tenant": "acme"})
				if err != nil {
					t.Fatal(err)
				}
				got, _ := url.PathUnescape(object.Key)
				decoded, _ := url.PathUnescape(key)
				if got != tt.want+decoded {
					t.Fatalf("object %q does not match namespace %q + key %q", got, tt.want, decoded)
				}
			}
		})
	}
}

func TestListingRejectsUnsafeMappingsAndHeadIgnoresKeyRules(t *testing.T) {
	for _, rw := range []config.RewriteRule{
		{StripKeyPrefix: "x"}, {StripPathPrefix: "/visible"}, {PrependKeyPrefix: "%oops"},
		{KeyTemplate: "{{ .Key }}/suffix", CompiledTemplate: template.Must(template.New("key").Parse("{{ .Key }}/suffix"))},
		{KeyTemplate: "{{ .Captures.missing }}/{{ .Key }}", CompiledTemplate: template.Must(template.New("key").Parse("{{ .Captures.missing }}/{{ .Key }}"))},
		{KeyTemplate: "{{ .Captures.bad }}{{ .Key }}", CompiledTemplate: template.Must(template.New("key").Parse("{{ .Captures.bad }}{{ .Key }}"))},
	} {
		ctx := &requestctx.Context{Method: "GET", Bucket: "visible", Query: url.Values{"list-type": {"2"}}}
		if _, err := New().Apply(ctx, rw, map[string]string{"bad": "%2"}); !errors.Is(err, ErrInvalidResult) {
			t.Fatalf("rule %+v: error = %v", rw, err)
		}
		ctx.Method = "HEAD"
		rw.Bucket = "store"
		result, err := New().Apply(ctx, rw, nil)
		if err != nil || result.Key != "" || result.Bucket != "store" || result.Namespace != nil {
			t.Fatalf("HeadBucket result = %+v, error = %v", result, err)
		}
	}
}
