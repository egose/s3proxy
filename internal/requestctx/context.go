package requestctx

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestquery"
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

var errNoAddressingMatch = errors.New("request does not match enabled listener addressing modes")

func IsNoAddressingMatch(err error) bool {
	return errors.Is(err, errNoAddressingMatch)
}

func FromRequest(r *http.Request, cfg config.Addressing) (*Context, error) {
	query, err := requestquery.Parse(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}
	escapedPath := requestEscapedPath(r)
	ctx := &Context{
		Host:    normalizeHost(r.Host),
		RawPath: escapedPath,
		Query:   query,
		Method:  r.Method,
		Headers: r.Header,
	}

	if bucket := VirtualBucket(r.Host, cfg); bucket != "" {
		ctx.AddressingMode = AddressingVirtualHosted
		ctx.Bucket = bucket
		ctx.Key = strings.TrimPrefix(escapedPath, "/")
	}

	if ctx.AddressingMode == "" && cfg.PathStyle {
		ctx.AddressingMode = AddressingPathStyle
		bucket, key := parsePathStyle(escapedPath)
		ctx.Bucket = bucket
		ctx.Key = key
	}

	if ctx.AddressingMode == "" {
		return nil, errNoAddressingMatch
	}

	return ctx, nil
}

func VirtualBucket(host string, cfg config.Addressing) string {
	if !cfg.VirtualHosted {
		return ""
	}
	host = normalizeHost(host)
	for _, suffix := range cfg.HostSuffixes {
		suffix = normalizeHost(suffix)
		if strings.HasSuffix(host, "."+suffix) {
			return strings.TrimSuffix(host, "."+suffix)
		}
	}
	return ""
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
		return strings.ToLower(strings.TrimSuffix(hostport, "."))
	}
	if strings.Count(hostport, ":") > 1 {
		return strings.ToLower(strings.TrimSuffix(hostport, "."))
	}
	return strings.ToLower(strings.TrimSuffix(hostport[:idx], "."))
}

func requestEscapedPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return "/"
	}
	if raw := r.URL.EscapedPath(); raw != "" {
		return raw
	}
	if r.URL.Path != "" {
		return r.URL.Path
	}
	return "/"
}
