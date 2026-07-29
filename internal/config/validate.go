package config

import (
	"fmt"
	"strings"
)

func Validate(rt *Runtime) error {
	if err := validateListener(rt.Listener); err != nil {
		return err
	}
	if err := validateAuth(rt.Auth); err != nil {
		return err
	}
	if err := validateTargets(rt.Targets); err != nil {
		return err
	}
	if err := validateParsers(rt.Parsers); err != nil {
		return err
	}
	if err := validateRoutes(rt.Routes, rt.Parsers, rt.Targets); err != nil {
		return err
	}
	if err := validateBuckets(rt.Buckets, rt.Routes); err != nil {
		return err
	}
	return nil
}

func validateListener(l Listener) error {
	if l.Address == "" {
		return fmt.Errorf("listener.http %q: address is required", l.Name)
	}
	if l.MaxHeaderBytes < 0 {
		return fmt.Errorf("listener.http %q: max_header_bytes must be >= 0", l.Name)
	}
	if l.ReplayBodyMaxBytes < 0 {
		return fmt.Errorf("listener.http %q: replay_body_max_bytes must be >= 0", l.Name)
	}
	if l.Timeouts.Read < 0 || l.Timeouts.ReadHeader < 0 || l.Timeouts.Idle < 0 || l.Timeouts.Write < 0 {
		return fmt.Errorf("listener.http %q: timeouts must be >= 0", l.Name)
	}
	if !l.Addressing.PathStyle && !l.Addressing.VirtualHosted {
		return fmt.Errorf("listener.http %q: at least one addressing mode must be enabled", l.Name)
	}
	if l.Addressing.VirtualHosted && len(l.Addressing.HostSuffixes) == 0 {
		return fmt.Errorf("listener.http %q: virtual_hosted requires at least one host_suffix", l.Name)
	}
	return nil
}

func validateAuth(a Auth) error {
	switch a.Mode {
	case AuthModeNone:
		if len(a.Clients) > 0 {
			return fmt.Errorf("auth %q: clients cannot be defined when mode is none", a.Name)
		}
	case AuthModeSigV4Static:
		accessKeys := make(map[string]string)
		for _, c := range a.Clients {
			if c.AccessKey == "" {
				return fmt.Errorf("auth %q: client %q has empty access_key", a.Name, c.Name)
			}
			if c.SecretKey == "" {
				return fmt.Errorf("auth %q: client %q has empty secret_key", a.Name, c.Name)
			}
			if existing, dup := accessKeys[c.AccessKey]; dup {
				return fmt.Errorf("auth %q: client %q and %q share the same access_key", a.Name, existing, c.Name)
			}
			accessKeys[c.AccessKey] = c.Name
		}
		if len(a.Clients) == 0 {
			return fmt.Errorf("auth %q: at least one client is required for sigv4_static mode", a.Name)
		}
	default:
		return fmt.Errorf("auth %q: invalid mode %q (must be none or sigv4_static)", a.Name, a.Mode)
	}
	return nil
}

func validateTargets(targets map[string]S3Target) error {
	for name, t := range targets {
		if t.Endpoint == "" {
			return fmt.Errorf("target.s3 %q: endpoint is required", name)
		}
		if t.EndpointURL == nil {
			return fmt.Errorf("target.s3 %q: parsed endpoint is missing", name)
		}
		if !t.EndpointURL.IsAbs() {
			return fmt.Errorf("target.s3 %q: endpoint must be an absolute URL", name)
		}
		if t.EndpointURL.Host == "" {
			return fmt.Errorf("target.s3 %q: endpoint host is required", name)
		}
		scheme := strings.ToLower(t.EndpointURL.Scheme)
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("target.s3 %q: endpoint scheme must be http or https", name)
		}
		if t.Region == "" {
			return fmt.Errorf("target.s3 %q: region is required", name)
		}
		if t.Credentials.AccessKey == "" {
			return fmt.Errorf("target.s3 %q: credentials access_key is empty", name)
		}
		if t.Credentials.SecretKey == "" {
			return fmt.Errorf("target.s3 %q: credentials secret_key is empty", name)
		}
		if t.Timeout < 0 {
			return fmt.Errorf("target.s3 %q: timeout must be >= 0", name)
		}
	}
	return nil
}

func validateParsers(parsers map[string]Parser) error {
	for name, p := range parsers {
		switch p.Kind {
		case ParserPathPrefix:
			if p.Prefix == "" {
				return fmt.Errorf("parser.path_prefix %q: prefix is required", name)
			}
		case ParserBucketExact:
			if p.Bucket == "" {
				return fmt.Errorf("parser.bucket_exact %q: bucket is required", name)
			}
		case ParserBucketRegex:
			if p.Pattern == "" {
				return fmt.Errorf("parser.bucket_regex %q: pattern is required", name)
			}
		case ParserHostSuffix:
			if p.Suffix == "" {
				return fmt.Errorf("parser.host_suffix %q: suffix is required", name)
			}
		default:
			return fmt.Errorf("parser %q: unknown kind %q", name, p.Kind)
		}
	}
	return nil
}

func validateRoutes(routes []Route, parsers map[string]Parser, targets map[string]S3Target) error {
	routeNames := make(map[string]bool)
	for _, r := range routes {
		if routeNames[r.Name] {
			return fmt.Errorf("duplicate route %q", r.Name)
		}
		routeNames[r.Name] = true

		if r.ParserRef == "" {
			return fmt.Errorf("route %q: parser is required", r.Name)
		}
		if _, ok := parsers[r.ParserRef]; !ok {
			return fmt.Errorf("route %q: unknown parser %q", r.Name, r.ParserRef)
		}
		if len(r.DestinationRefs) == 0 {
			return fmt.Errorf("route %q: at least one destination is required", r.Name)
		}
		for _, d := range r.DestinationRefs {
			if _, ok := targets[d]; !ok {
				return fmt.Errorf("route %q: unknown destination %q", r.Name, d)
			}
		}
		switch r.Dispatch {
		case DispatchFirst, DispatchAll:
		default:
			return fmt.Errorf("route %q: invalid dispatch %q", r.Name, r.Dispatch)
		}
		switch r.OnMatch {
		case MatchStop:
		case MatchContinue:
			if !routeSupportsContinue(r) {
				return fmt.Errorf("route %q: on_match %q is only implemented for write-only routes", r.Name, r.OnMatch)
			}
		default:
			return fmt.Errorf("route %q: invalid on_match %q", r.Name, r.OnMatch)
		}
		switch r.ReadPreference {
		case ReadFirst, ReadRandom, ReadHash, ReadOrderedFailover:
		default:
			return fmt.Errorf("route %q: invalid read_preference %q", r.Name, r.ReadPreference)
		}
		if len(r.Operations) == 0 {
			return fmt.Errorf("route %q: at least one operation is required", r.Name)
		}
		for _, op := range r.Operations {
			if !isValidOperation(op) {
				return fmt.Errorf("route %q: unsupported operation %q", r.Name, op)
			}
			if op == "CopyObject" {
				return fmt.Errorf("route %q: operation %q is not implemented", r.Name, op)
			}
			if r.Dispatch == DispatchAll && !supportsDispatchAll(op) {
				return fmt.Errorf("route %q: dispatch %q is only implemented for PutObject and DeleteObject, got %q", r.Name, r.Dispatch, op)
			}
		}
		if r.Dispatch == DispatchAll && r.ReadPreference != ReadFirst {
			return fmt.Errorf("route %q: read_preference is ignored for dispatch=all", r.Name)
		}
	}
	return nil
}

func validateBuckets(buckets []VirtualBucket, routes []Route) error {
	routeNames := make(map[string]bool)
	for _, r := range routes {
		routeNames[r.Name] = true
	}
	visibleNames := make(map[string]string)
	for _, b := range buckets {
		if b.VisibleName == "" {
			return fmt.Errorf("bucket %q: visible_name is required", b.Name)
		}
		if existing, dup := visibleNames[b.VisibleName]; dup {
			return fmt.Errorf("duplicate visible bucket name %q (from %q and %q)", b.VisibleName, existing, b.Name)
		}
		visibleNames[b.VisibleName] = b.Name
		if !routeNames[b.RouteRef] {
			return fmt.Errorf("bucket %q: unknown route ref %q", b.Name, b.RouteRef)
		}
	}
	return nil
}

func isValidOperation(op string) bool {
	switch op {
	case "GetObject", "HeadObject", "PutObject", "DeleteObject",
		"HeadBucket", "ListObjectsV2", "ListBuckets", "CopyObject":
		return true
	default:
		return false
	}
}

func routeSupportsContinue(r Route) bool {
	if len(r.Operations) == 0 {
		return false
	}
	for _, op := range r.Operations {
		switch op {
		case "PutObject", "DeleteObject":
		default:
			return false
		}
	}
	return true
}

func supportsDispatchAll(op string) bool {
	switch op {
	case "PutObject", "DeleteObject":
		return true
	default:
		return false
	}
}
