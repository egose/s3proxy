package config

import (
	"strconv"
	"strings"
	"testing"
)

func TestLoadListingRewriteValidation(t *testing.T) {
	base := strings.Replace(exampleConfig, `operations   = ["GetObject", "PutObject"]`, `operations   = ["GetObject", "HeadBucket", "ListObjectsV2"]`, 1)
	base = strings.Replace(base, `strip_path_prefix  = "/images"`, "", 1)
	for _, tt := range []struct {
		name, rule string
		valid      bool
	}{
		{"bucket only", "", true},
		{"prepend", `prepend_key_prefix = "assets/"`, true},
		{"escaped prepend", `prepend_key_prefix = "assets%20%252F/"`, true},
		{"bad escape", `prepend_key_prefix = "assets%/"`, false},
		{"strip path", `strip_path_prefix = "/images"`, false},
		{"strip key", `strip_key_prefix = "assets/"`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			content := strings.Replace(base, `prepend_key_prefix = "assets/"`, tt.rule, 1)
			_, err := LoadFile(writeTmpConfig(t, content))
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v err=%v", tt.valid, err)
			}
			if err != nil && !strings.Contains(err.Error(), `route "images_rw": ListObjectsV2`) {
				t.Fatalf("missing route diagnostic: %v", err)
			}
		})
	}
	for _, tt := range []struct {
		tmpl  string
		valid bool
	}{
		{"{{ .Key }}", true},
		{"{{ .Captures.tenant }}/{{ .Key }}", true},
		{"{{ .Bucket }}/{{ .Captures.tenant }}/literal{{ .Key }}", true},
		{"{{ .Key }}/suffix", false},
		{"{{ .Key }}{{ .Key }}", false},
		{"{{ .Key }} ", false},
		{"no key", false},
		{"{{ if .Bucket }}prefix{{ end }}{{ .Key }}", false},
		{"{{ printf \"%s\" .Key }}", false},
		{"{{ .Key | printf \"%s\" }}", false},
		{"{{ $x := .Bucket }}{{ .Key }}", false},
		{"{{ index .Captures \"tenant\" }}/{{ .Key }}", false},
		{"{{ .Captures }}/{{ .Key }}", false},
		{"{{ define \"other\" }}x{{ end }}{{ .Key }}", false},
		{"bad%GG/{{ .Key }}", false},
	} {
		t.Run(tt.tmpl, func(t *testing.T) {
			content := strings.Replace(base, `prepend_key_prefix = "assets/"`, "key_template = "+strconv.Quote(tt.tmpl), 1)
			_, err := LoadFile(writeTmpConfig(t, content))
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v err=%v", tt.valid, err)
			}
			objectOnly := strings.Replace(content, `"GetObject", "HeadBucket", "ListObjectsV2"`, `"GetObject", "HeadBucket"`, 1)
			if _, err := LoadFile(writeTmpConfig(t, objectOnly)); err != nil {
				t.Fatalf("object/head route must retain general templates: %v", err)
			}
		})
	}
}
