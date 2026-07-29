package config

import (
	"net/url"
	"regexp"
	"text/template"
	"time"
)

type Runtime struct {
	Listener Listener
	Auth     Auth
	Targets  map[string]S3Target
	Parsers  map[string]Parser
	Routes   []Route
	Buckets  []VirtualBucket
}

type Listener struct {
	Name               string
	Address            string
	Addressing         Addressing
	MaxHeaderBytes     int
	ReplayBodyMaxBytes int64
	Timeouts           Timeouts
}

type Addressing struct {
	PathStyle     bool
	VirtualHosted bool
	HostSuffixes  []string
}

type Timeouts struct {
	Read       time.Duration
	ReadHeader time.Duration
	Idle       time.Duration
	Write      time.Duration
}

type AuthMode string

const (
	AuthModeNone        AuthMode = "none"
	AuthModeSigV4Static AuthMode = "sigv4_static"
)

type DispatchMode string

const (
	DispatchFirst DispatchMode = "first"
	DispatchAll   DispatchMode = "all"
)

type MatchMode string

const (
	MatchStop     MatchMode = "stop"
	MatchContinue MatchMode = "continue"
)

type ReadPreference string

const (
	ReadFirst           ReadPreference = "first"
	ReadRandom          ReadPreference = "random"
	ReadHash            ReadPreference = "hash"
	ReadOrderedFailover ReadPreference = "ordered_failover"
)

type ParserKind string

const (
	ParserPathPrefix  ParserKind = "path_prefix"
	ParserBucketExact ParserKind = "bucket_exact"
	ParserBucketRegex ParserKind = "bucket_regex"
	ParserHostSuffix  ParserKind = "host_suffix"
)

type Auth struct {
	Name    string
	Mode    AuthMode
	Clients map[string]Client
}

type Client struct {
	Name           string
	AccessKey      string
	SecretKey      string
	AllowRoutes    []string
	AllowOps       []string
	VisibleBuckets []string
}

type StaticCredential struct {
	Name      string
	AccessKey string
	SecretKey string
}

type S3Target struct {
	Name           string
	Endpoint       string
	EndpointURL    *url.URL
	Region         string
	ForcePathStyle bool
	Timeout        time.Duration
	Credentials    StaticCredential
}

type Parser struct {
	Name    string
	Kind    ParserKind
	Prefix  string
	Bucket  string
	Pattern string
	Suffix  string
	Regex   *regexp.Regexp
}

type Route struct {
	Name            string
	ParserRef       string
	Operations      []string
	DestinationRefs []string
	Dispatch        DispatchMode
	OnMatch         MatchMode
	ReadPreference  ReadPreference
	Rewrite         RewriteRule
}

type RewriteRule struct {
	StripPathPrefix  string
	StripKeyPrefix   string
	PrependKeyPrefix string
	Bucket           string
	KeyTemplate      string
	CompiledTemplate *template.Template
}

type VirtualBucket struct {
	Name        string
	VisibleName string
	RouteRef    string
}
