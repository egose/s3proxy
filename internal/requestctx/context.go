package requestctx

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/egose/s3proxy/internal/config"
)

type AddressingMode string

const (
	AddressingPathStyle     AddressingMode = "path_style"
	AddressingVirtualHosted AddressingMode = "virtual_hosted"
)

type Context struct {
	Host           string
	RawPath        string
	Bucket         string
	Key            string
	Query          url.Values
	Method         string
	Headers        http.Header
	AddressingMode AddressingMode
	Captures       map[string]string
}

func FromRequest(r *http.Request, cfg config.Addressing) (*Context, error) {
	ctx := &Context{
		Host:    normalizeHost(r.Host),
		RawPath: r.URL.Path,
		Query:   r.URL.Query(),
		Method:  r.Method,
		Headers: r.Header,
	}

	if cfg.VirtualHosted && len(cfg.HostSuffixes) > 0 {
		for _, suffix := range cfg.HostSuffixes {
			if strings.HasSuffix(ctx.Host, "."+suffix) {
				bucket := strings.TrimSuffix(ctx.Host, "."+suffix)
				if bucket != "" {
					ctx.AddressingMode = AddressingVirtualHosted
					ctx.Bucket = bucket
					ctx.Key = strings.TrimPrefix(r.URL.Path, "/")
				}
				break
			}
		}
	}

	if ctx.AddressingMode == "" && cfg.PathStyle {
		ctx.AddressingMode = AddressingPathStyle
		bucket, key := parsePathStyle(r.URL.Path)
		ctx.Bucket = bucket
		ctx.Key = key
	}

	if ctx.AddressingMode == "" {
		ctx.AddressingMode = AddressingPathStyle
		bucket, key := parsePathStyle(r.URL.Path)
		ctx.Bucket = bucket
		ctx.Key = key
	}

	return ctx, nil
}

func parsePathStyle(path string) (bucket, key string) {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "", ""
	}
	idx := strings.Index(path, "/")
	if idx == -1 {
		return path, ""
	}
	return path[:idx], path[idx+1:]
}

func normalizeHost(hostport string) string {
	idx := strings.LastIndex(hostport, ":")
	if idx == -1 {
		return hostport
	}
	if strings.Count(hostport, ":") > 1 {
		return hostport
	}
	return hostport[:idx]
}
