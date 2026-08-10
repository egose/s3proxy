package rewrite

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
)

type Result struct {
	Bucket string
	Key    string
}

type Engine interface {
	Apply(ctx *requestctx.Context, rule config.RewriteRule, captures map[string]string) (Result, error)
}

func New() Engine {
	return &engine{}
}

type engine struct{}

type templateData struct {
	Bucket   string
	Key      string
	Captures map[string]string
}

func (e *engine) Apply(ctx *requestctx.Context, rw config.RewriteRule, captures map[string]string) (Result, error) {
	bucket := ctx.Bucket
	key := ctx.Key

	if rw.StripPathPrefix != "" && pathPrefixMatches(ctx.RawPath, rw.StripPathPrefix) {
		remaining := strings.TrimPrefix(ctx.RawPath, rw.StripPathPrefix)
		remaining = strings.TrimPrefix(remaining, "/")
		if ctx.Key != "" {
			key = remaining
		}
	}

	if rw.StripKeyPrefix != "" && strings.HasPrefix(key, rw.StripKeyPrefix) {
		key = strings.TrimPrefix(key, rw.StripKeyPrefix)
	}

	if rw.PrependKeyPrefix != "" {
		key = cleanJoinedKey(rw.PrependKeyPrefix, key)
	}

	if rw.Bucket != "" {
		bucket = rw.Bucket
	}

	if rw.KeyTemplate != "" {
		data := templateData{
			Bucket:   bucket,
			Key:      key,
			Captures: captures,
		}
		if rw.CompiledTemplate == nil {
			return Result{}, fmt.Errorf("invalid key_template: compiled template missing")
		}
		var buf bytes.Buffer
		if err := rw.CompiledTemplate.Execute(&buf, data); err != nil {
			return Result{}, fmt.Errorf("key_template execution failed: %w", err)
		}
		key = buf.String()
	}

	if bucket == "" {
		return Result{}, fmt.Errorf("rewrite produced empty bucket")
	}

	return Result{Bucket: bucket, Key: key}, nil
}

func pathPrefixMatches(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func cleanJoinedKey(prefix, key string) string {
	prefix = strings.TrimSuffix(prefix, "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return prefix
	}
	return prefix + "/" + key
}
