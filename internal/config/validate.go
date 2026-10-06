package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/egose/s3proxy/internal/namespace"
	"github.com/egose/s3proxy/internal/s3op"
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
	if err := validateAuthPolicyRefs(rt.Auth, rt.Routes, rt.Buckets); err != nil {
		return err
	}
	return nil
}

func validateListener(l Listener) error {
	if l.Address == "" {
		return fmt.Errorf("listener.http %q: address is required", l.Name)
	}
	_, port, err := net.SplitHostPort(l.Address)
	if err != nil || strings.Contains(l.Address, "://") {
		return fmt.Errorf("listener.http %q: address must use TCP host:port syntax", l.Name)
	}
	if !listenerPortInRange(port) {
		return fmt.Errorf("listener.http %q: address numeric port must be between 0 and 65535", l.Name)
	}
	if l.MaxHeaderBytes < 0 {
		return fmt.Errorf("listener.http %q: max_header_bytes must be >= 0", l.Name)
	}
	if l.ReplayBodyMaxBytes < 0 {
		return fmt.Errorf("listener.http %q: replay_body_max_bytes must be >= 0", l.Name)
	}
	if l.ReplayBodyAggregateMaxBytes < 0 {
		return fmt.Errorf("listener.http %q: replay_body_aggregate_max_bytes must be >= 0", l.Name)
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

func listenerPortInRange(port string) bool {
	negative := strings.HasPrefix(port, "-")
	if negative || strings.HasPrefix(port, "+") {
		port = port[1:]
	}
	var n uint32
	outOfRange := false
	for _, digit := range port {
		if digit < '0' || digit > '9' {
			return true
		}
		if n >= 1<<30 {
			return false
		}
		n *= 10
		next := n + uint32(digit-'0')
		if next < n {
			return false
		}
		n = next
		outOfRange = outOfRange || n > 65535
	}
	return !outOfRange && (!negative || n == 0)
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
		return fmt.Errorf("auth %q: invalid mode (must be none or sigv4_static)", a.Name)
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
			return fmt.Errorf("route %q: unknown parser ref", r.Name)
		}
		if len(r.DestinationRefs) == 0 {
			return fmt.Errorf("route %q: at least one destination is required", r.Name)
		}
		seenDestinations := make(map[string]bool, len(r.DestinationRefs))
		for _, d := range r.DestinationRefs {
			targetName := stripRefPrefix(d)
			if seenDestinations[targetName] {
				return fmt.Errorf("route %q: destinations contain duplicate target", r.Name)
			}
			seenDestinations[targetName] = true
			if _, ok := targets[targetName]; !ok {
				return fmt.Errorf("route %q: unknown destination in destinations", r.Name)
			}
		}
		switch r.Dispatch {
		case DispatchFirst, DispatchAll:
		default:
			return fmt.Errorf("route %q: invalid dispatch (must be first or all)", r.Name)
		}
		switch r.OnMatch {
		case MatchStop:
		case MatchContinue:
			if !routeSupportsContinue(r) {
				return fmt.Errorf("route %q: on_match continue is only implemented for write-only routes", r.Name)
			}
		default:
			return fmt.Errorf("route %q: invalid on_match (must be stop or continue)", r.Name)
		}
		switch r.ReadPreference {
		case ReadFirst, ReadRandom, ReadHash, ReadOrderedFailover:
		default:
			return fmt.Errorf("route %q: invalid read_preference (must be first, random, hash or ordered_failover)", r.Name)
		}
		if len(r.Operations) == 0 {
			return fmt.Errorf("route %q: at least one operation is required", r.Name)
		}
		seenOperations := make(map[string]bool, len(r.Operations))
		for _, op := range r.Operations {
			if seenOperations[op] {
				return fmt.Errorf("route %q: operations contain duplicate entry", r.Name)
			}
			seenOperations[op] = true
			if op == string(s3op.ListObjectsV2) {
				if err := ValidateListingRewrite(r.Rewrite); err != nil {
					return fmt.Errorf("route %q: ListObjectsV2: %w", r.Name, err)
				}
			}
			if !s3op.IsConfigurable(op) {
				return fmt.Errorf("route %q: unsupported operation in operations", r.Name)
			}
			if r.Dispatch == DispatchAll && !supportsDispatchAllRouteOperation(op) {
				return fmt.Errorf("route %q: dispatch all does not support a configured write operation", r.Name)
			}
		}
		if r.Dispatch == DispatchAll && !routeHasFanoutWrite(r) {
			return fmt.Errorf("route %q: dispatch all requires PutObject or DeleteObject", r.Name)
		}
		if r.Dispatch == DispatchAll && !routeHasRead(r) && r.ReadPreference != ReadFirst {
			return fmt.Errorf("route %q: read_preference is ignored for dispatch=all", r.Name)
		}
	}
	return nil
}

func ValidateListingRewrite(rw RewriteRule) error {
	if rw.StripPathPrefix != "" || rw.StripKeyPrefix != "" {
		return fmt.Errorf("%w: strip_path_prefix and strip_key_prefix are unsupported", namespace.ErrMapping)
	}
	if _, err := url.PathUnescape(rw.PrependKeyPrefix); err != nil {
		return fmt.Errorf("%w: prepend_key_prefix must use valid percent escapes", namespace.ErrMapping)
	}
	if rw.KeyTemplate != "" {
		if err := namespace.ValidateTemplate(rw.CompiledTemplate); err != nil {
			return fmt.Errorf("key_template: %w", err)
		}
		if prefix, err := namespace.RawTemplatePrefix(rw.CompiledTemplate, rw.Bucket, nil); err == nil {
			if _, err := namespace.New("", prefix+rw.PrependKeyPrefix); err != nil {
				return fmt.Errorf("key_template and prepend_key_prefix: %w", err)
			}
		}
	}
	return nil
}

func validateBuckets(buckets []VirtualBucket, routes []Route) error {
	routeNames := make(map[string]bool)
	for _, r := range routes {
		routeNames[r.Name] = true
	}
	bucketNames := make(map[string]bool)
	visibleNames := make(map[string]string)
	for _, b := range buckets {
		if bucketNames[b.Name] {
			return fmt.Errorf("duplicate bucket %q", b.Name)
		}
		bucketNames[b.Name] = true
		if b.VisibleName == "" {
			return fmt.Errorf("bucket %q: visible_name is required", b.Name)
		}
		if existing, dup := visibleNames[b.VisibleName]; dup {
			return fmt.Errorf("bucket %q: duplicate visible_name (also in bucket %q)", b.Name, existing)
		}
		visibleNames[b.VisibleName] = b.Name
		if !routeNames[b.RouteRef] {
			return fmt.Errorf("bucket %q: unknown route ref", b.Name)
		}
	}
	return nil
}

func validateAuthPolicyRefs(a Auth, routes []Route, buckets []VirtualBucket) error {
	if a.Mode != AuthModeSigV4Static {
		return nil
	}
	routeNames := make(map[string]bool, len(routes))
	for _, r := range routes {
		routeNames[r.Name] = true
	}
	bucketNames := make(map[string]bool, len(buckets))
	for _, b := range buckets {
		bucketNames[b.VisibleName] = true
	}
	for _, c := range a.Clients {
		if err := validateClientStringRefs(a.Name, c.Name, "allow_routes", c.AllowRoutes, routeNames); err != nil {
			return err
		}
		if err := validateClientOps(a.Name, c); err != nil {
			return err
		}
		if err := validateClientStringRefs(a.Name, c.Name, "visible_buckets", c.VisibleBuckets, bucketNames); err != nil {
			return err
		}
	}
	return nil
}

func validateClientStringRefs(authName, clientName, field string, refs []string, known map[string]bool) error {
	seen := make(map[string]bool, len(refs))
	for _, ref := range refs {
		if seen[ref] {
			return fmt.Errorf("auth %q: client %q: %s contains duplicate entry", authName, clientName, field)
		}
		seen[ref] = true
		if ref == "*" {
			continue
		}
		if !known[ref] {
			return fmt.Errorf("auth %q: client %q: %s references unknown %s", authName, clientName, field, policyRefKind(field))
		}
	}
	return nil
}

func validateClientOps(authName string, c Client) error {
	seen := make(map[string]bool, len(c.AllowOps))
	for _, op := range c.AllowOps {
		if seen[op] {
			return fmt.Errorf("auth %q: client %q: allow_ops contains duplicate entry", authName, c.Name)
		}
		seen[op] = true
		if op == "*" {
			continue
		}
		if !s3op.IsConfigurable(op) {
			return fmt.Errorf("auth %q: client %q: allow_ops contains unsupported operation", authName, c.Name)
		}
	}
	return nil
}

func policyRefKind(field string) string {
	switch field {
	case "allow_routes":
		return "route"
	case "visible_buckets":
		return "bucket"
	default:
		return "reference"
	}
}

func routeSupportsContinue(r Route) bool {
	if len(r.Operations) == 0 {
		return false
	}
	for _, op := range r.Operations {
		if !s3op.IsWrite(s3op.Operation(op)) {
			return false
		}
	}
	return true
}

func supportsDispatchAll(op string) bool {
	return s3op.SupportsFanout(s3op.Operation(op))
}

func supportsDispatchAllRouteOperation(op string) bool {
	return s3op.IsRead(s3op.Operation(op)) || supportsDispatchAll(op)
}

func routeHasFanoutWrite(r Route) bool {
	for _, op := range r.Operations {
		if supportsDispatchAll(op) {
			return true
		}
	}
	return false
}

func routeHasRead(r Route) bool {
	for _, op := range r.Operations {
		if s3op.IsRead(s3op.Operation(op)) {
			return true
		}
	}
	return false
}
