package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func LoadFile(path string) (*Runtime, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	return Load(src, path)
}

func Load(src []byte, filename string) (*Runtime, error) {
	src = trimLeadingNewlines(src)
	expanded := expandEnvCalls(src)

	file, diags := hclsyntax.ParseConfig(expanded, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse config %s: %s", filename, diags.Error())
	}

	var raw rawFile
	if d := gohcl.DecodeBody(file.Body, nil, &raw); d.HasErrors() {
		return nil, fmt.Errorf("decode config %s: %s", filename, d.Error())
	}

	rt, err := buildRuntime(&raw)
	if err != nil {
		return nil, err
	}
	if err := Validate(rt); err != nil {
		return nil, err
	}
	return rt, nil
}

// expandEnvCalls replaces env("VAR") calls in the HCL source with the literal
// environment value, so credential secret injection works without a full HCL
// evaluation context.
func expandEnvCalls(src []byte) []byte {
	re := regexp.MustCompile(`env\("([^"]+)"\)`)
	return re.ReplaceAllFunc(src, func(match []byte) []byte {
		sub := re.FindSubmatch(match)
		if len(sub) < 2 {
			return []byte(`""`)
		}
		val := os.Getenv(string(sub[1]))
		return []byte(strconv.Quote(val))
	})
}

func trimLeadingNewlines(src []byte) []byte {
	for len(src) > 0 && (src[0] == '\n' || src[0] == '\r' || src[0] == ' ' || src[0] == '\t') {
		src = src[1:]
	}
	return src
}

type rawFile struct {
	Listeners   []rawListener   `hcl:"listener,block"`
	Auth        []rawAuth       `hcl:"auth,block"`
	Credentials []rawCredential `hcl:"credential,block"`
	Targets     []rawTarget     `hcl:"target,block"`
	Parsers     []rawParser     `hcl:"parser,block"`
	Routes      []rawRoute      `hcl:"route,block"`
	Buckets     []rawBucket     `hcl:"bucket,block"`
}

type rawListener struct {
	Type                        string        `hcl:"type,label"`
	Name                        string        `hcl:"name,label"`
	Address                     string        `hcl:"address"`
	MaxHeaderBytes              int           `hcl:"max_header_bytes,optional"`
	ReplayBodyMaxBytes          int64         `hcl:"replay_body_max_bytes,optional"`
	ReplayBodyAggregateMaxBytes int64         `hcl:"replay_body_aggregate_max_bytes,optional"`
	Addressing                  rawAddressing `hcl:"addressing,block"`
	Timeouts                    *rawTimeouts  `hcl:"timeouts,block"`
}

type rawAddressing struct {
	PathStyle     bool     `hcl:"path_style"`
	VirtualHosted bool     `hcl:"virtual_hosted,optional"`
	HostSuffixes  []string `hcl:"host_suffixes,optional"`
}

type rawTimeouts struct {
	Read       string `hcl:"read,optional"`
	ReadHeader string `hcl:"read_header,optional"`
	Idle       string `hcl:"idle,optional"`
	Write      string `hcl:"write,optional"`
}

type rawAuth struct {
	Name    string      `hcl:"name,label"`
	Mode    string      `hcl:"mode"`
	Clients []rawClient `hcl:"client,block"`
}

type rawClient struct {
	Name           string   `hcl:"name,label"`
	AccessKey      string   `hcl:"access_key"`
	SecretKey      string   `hcl:"secret_key"`
	AllowRoutes    []string `hcl:"allow_routes,optional"`
	AllowOps       []string `hcl:"allow_ops,optional"`
	VisibleBuckets []string `hcl:"visible_buckets,optional"`
}

type rawCredential struct {
	Type      string `hcl:"type,label"`
	Name      string `hcl:"name,label"`
	AccessKey string `hcl:"access_key"`
	SecretKey string `hcl:"secret_key"`
}

type rawTarget struct {
	Type           string `hcl:"type,label"`
	Name           string `hcl:"name,label"`
	Endpoint       string `hcl:"endpoint"`
	Region         string `hcl:"region"`
	ForcePathStyle bool   `hcl:"force_path_style,optional"`
	Timeout        string `hcl:"timeout,optional"`
	Credentials    string `hcl:"credentials"`
}

type rawParser struct {
	Type    string `hcl:"type,label"`
	Name    string `hcl:"name,label"`
	Prefix  string `hcl:"prefix,optional"`
	Bucket  string `hcl:"bucket,optional"`
	Pattern string `hcl:"pattern,optional"`
	Suffix  string `hcl:"suffix,optional"`
}

type rawRoute struct {
	Name           string      `hcl:"name,label"`
	Parser         string      `hcl:"parser"`
	Operations     []string    `hcl:"operations"`
	Destinations   []string    `hcl:"destinations"`
	Dispatch       string      `hcl:"dispatch"`
	OnMatch        string      `hcl:"on_match"`
	ReadPreference string      `hcl:"read_preference,optional"`
	Rewrite        *rawRewrite `hcl:"rewrite,block"`
}

type rawRewrite struct {
	StripPathPrefix  string `hcl:"strip_path_prefix,optional"`
	StripKeyPrefix   string `hcl:"strip_key_prefix,optional"`
	PrependKeyPrefix string `hcl:"prepend_key_prefix,optional"`
	Bucket           string `hcl:"bucket,optional"`
	KeyTemplate      string `hcl:"key_template,optional"`
}

type rawBucket struct {
	Name        string `hcl:"name,label"`
	VisibleName string `hcl:"visible_name"`
	Route       string `hcl:"route"`
}

func buildRuntime(raw *rawFile) (*Runtime, error) {
	rt := &Runtime{
		Targets: make(map[string]S3Target),
		Parsers: make(map[string]Parser),
	}

	// Listener
	if len(raw.Listeners) == 0 {
		return nil, fmt.Errorf("no listener block defined")
	}
	if len(raw.Listeners) > 1 {
		return nil, fmt.Errorf("only one listener block is supported")
	}
	l := raw.Listeners[0]
	if l.Type != "http" {
		return nil, fmt.Errorf("unsupported listener type %q (only \"http\" is supported)", l.Type)
	}
	listener := Listener{
		Name:                        l.Name,
		Address:                     l.Address,
		MaxHeaderBytes:              l.MaxHeaderBytes,
		ReplayBodyMaxBytes:          l.ReplayBodyMaxBytes,
		ReplayBodyAggregateMaxBytes: l.ReplayBodyAggregateMaxBytes,
		Addressing: Addressing{
			PathStyle:     l.Addressing.PathStyle,
			VirtualHosted: l.Addressing.VirtualHosted,
			HostSuffixes:  l.Addressing.HostSuffixes,
		},
	}
	if l.Timeouts != nil {
		var err error
		listener.Timeouts, err = parseTimeouts(l.Timeouts)
		if err != nil {
			return nil, err
		}
	}
	rt.Listener = listener

	// Auth
	if len(raw.Auth) == 0 {
		return nil, fmt.Errorf("no auth block defined")
	}
	if len(raw.Auth) > 1 {
		return nil, fmt.Errorf("only one auth block is supported")
	}
	a := raw.Auth[0]
	auth := Auth{
		Name:    a.Name,
		Mode:    AuthMode(a.Mode),
		Clients: make(map[string]Client),
	}
	for _, c := range a.Clients {
		if _, dup := auth.Clients[c.Name]; dup {
			return nil, fmt.Errorf("duplicate client %q in auth %q", c.Name, a.Name)
		}
		auth.Clients[c.Name] = Client{
			Name:           c.Name,
			AccessKey:      c.AccessKey,
			SecretKey:      c.SecretKey,
			AllowRoutes:    stripRefList(c.AllowRoutes),
			AllowOps:       c.AllowOps,
			VisibleBuckets: c.VisibleBuckets,
		}
	}
	rt.Auth = auth

	// Credentials
	creds := make(map[string]StaticCredential)
	for _, cr := range raw.Credentials {
		if cr.Type != "static" {
			return nil, fmt.Errorf("unsupported credential type %q (only \"static\" is supported)", cr.Type)
		}
		if _, dup := creds[cr.Name]; dup {
			return nil, fmt.Errorf("duplicate credential.static %q", cr.Name)
		}
		creds[cr.Name] = StaticCredential{
			Name:      cr.Name,
			AccessKey: cr.AccessKey,
			SecretKey: cr.SecretKey,
		}
	}

	// Targets
	for _, t := range raw.Targets {
		if t.Type != "s3" {
			return nil, fmt.Errorf("unsupported target type %q (only \"s3\" is supported)", t.Type)
		}
		if _, dup := rt.Targets[t.Name]; dup {
			return nil, fmt.Errorf("duplicate target.s3 %q", t.Name)
		}
		cred, err := resolveCredentialRef(t.Credentials, creds)
		if err != nil {
			return nil, fmt.Errorf("target.s3 %q: %w", t.Name, err)
		}
		endpointURL, err := url.Parse(t.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("target.s3 %q: invalid endpoint: %w", t.Name, err)
		}
		timeout, err := parseOptionalDuration(t.Timeout)
		if err != nil {
			return nil, fmt.Errorf("target.s3 %q: invalid timeout: %w", t.Name, err)
		}
		rt.Targets[t.Name] = S3Target{
			Name:           t.Name,
			Endpoint:       t.Endpoint,
			EndpointURL:    endpointURL,
			Region:         t.Region,
			ForcePathStyle: t.ForcePathStyle,
			Timeout:        timeout,
			Credentials:    cred,
		}
	}

	// Parsers
	for _, p := range raw.Parsers {
		if _, dup := rt.Parsers[p.Name]; dup {
			return nil, fmt.Errorf("duplicate parser %q", p.Name)
		}
		kind := ParserKind(p.Type)
		var compiled *regexp.Regexp
		if kind == ParserBucketRegex {
			var err error
			compiled, err = regexp.Compile(p.Pattern)
			if err != nil {
				return nil, fmt.Errorf("parser.bucket_regex %q: invalid pattern: %w", p.Name, err)
			}
		}
		rt.Parsers[p.Name] = Parser{
			Name:    p.Name,
			Kind:    kind,
			Prefix:  p.Prefix,
			Bucket:  p.Bucket,
			Pattern: p.Pattern,
			Suffix:  p.Suffix,
			Regex:   compiled,
		}
	}

	// Routes
	for _, r := range raw.Routes {
		parserName := stripRefPrefix(r.Parser)
		if _, ok := rt.Parsers[parserName]; !ok {
			return nil, fmt.Errorf("route %q: unknown parser ref %q", r.Name, r.Parser)
		}
		for _, d := range r.Destinations {
			if _, ok := rt.Targets[stripRefPrefix(d)]; !ok {
				return nil, fmt.Errorf("route %q: unknown destination ref %q", r.Name, d)
			}
		}
		route := Route{
			Name:            r.Name,
			ParserRef:       parserName,
			Operations:      r.Operations,
			DestinationRefs: stripRefList(r.Destinations),
			Dispatch:        DispatchMode(r.Dispatch),
			OnMatch:         MatchMode(r.OnMatch),
			ReadPreference:  ReadPreference(r.ReadPreference),
		}
		if r.ReadPreference == "" {
			route.ReadPreference = ReadFirst
		}
		if r.Rewrite != nil {
			var compiledTemplate *template.Template
			if r.Rewrite.KeyTemplate != "" {
				var err error
				compiledTemplate, err = template.New("key").Option("missingkey=error").Parse(r.Rewrite.KeyTemplate)
				if err != nil {
					return nil, fmt.Errorf("route %q: invalid key_template: %w", r.Name, err)
				}
			}
			route.Rewrite = RewriteRule{
				StripPathPrefix:  r.Rewrite.StripPathPrefix,
				StripKeyPrefix:   r.Rewrite.StripKeyPrefix,
				PrependKeyPrefix: r.Rewrite.PrependKeyPrefix,
				Bucket:           r.Rewrite.Bucket,
				KeyTemplate:      r.Rewrite.KeyTemplate,
				CompiledTemplate: compiledTemplate,
			}
		}
		rt.Routes = append(rt.Routes, route)
	}

	// Buckets
	for _, b := range raw.Buckets {
		rt.Buckets = append(rt.Buckets, VirtualBucket{
			Name:        b.Name,
			VisibleName: b.VisibleName,
			RouteRef:    stripRefPrefix(b.Route),
		})
	}

	return rt, nil
}

func parseTimeouts(t *rawTimeouts) (Timeouts, error) {
	out := Timeouts{}
	if t.Read != "" {
		d, err := time.ParseDuration(t.Read)
		if err != nil {
			return out, fmt.Errorf("invalid read timeout: %w", err)
		}
		out.Read = d
	}
	if t.ReadHeader != "" {
		d, err := time.ParseDuration(t.ReadHeader)
		if err != nil {
			return out, fmt.Errorf("invalid read_header timeout: %w", err)
		}
		out.ReadHeader = d
	}
	if t.Idle != "" {
		d, err := time.ParseDuration(t.Idle)
		if err != nil {
			return out, fmt.Errorf("invalid idle timeout: %w", err)
		}
		out.Idle = d
	}
	if t.Write != "" {
		d, err := time.ParseDuration(t.Write)
		if err != nil {
			return out, fmt.Errorf("invalid write timeout: %w", err)
		}
		out.Write = d
	}
	return out, nil
}

func parseOptionalDuration(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	return time.ParseDuration(raw)
}

func resolveCredentialRef(ref string, creds map[string]StaticCredential) (StaticCredential, error) {
	name := stripRefPrefix(ref)
	c, ok := creds[name]
	if !ok {
		return StaticCredential{}, fmt.Errorf("unknown credential.static ref %q", ref)
	}
	return c, nil
}

func stripRefPrefix(ref string) string {
	idx := strings.LastIndex(ref, ".")
	if idx == -1 {
		return ref
	}
	return ref[idx+1:]
}

func stripRefList(refs []string) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, stripRefPrefix(r))
	}
	return out
}
