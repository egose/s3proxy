package router

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/s3ops"
)

type Match struct {
	Route         config.Route
	Destinations  []config.S3Target
	EffectiveRead *config.S3Target
	Captures      map[string]string
}

type RouteResolver interface {
	Resolve(ctx *requestctx.Context, op s3ops.Operation) ([]Match, error)
}

func NewResolver(rt *config.Runtime) RouteResolver {
	return &resolver{
		routes:  rt.Routes,
		parsers: rt.Parsers,
		targets: rt.Targets,
	}
}

type resolver struct {
	routes  []config.Route
	parsers map[string]config.Parser
	targets map[string]config.S3Target
}

func (r *resolver) Resolve(ctx *requestctx.Context, op s3ops.Operation) ([]Match, error) {
	var matches []Match
	for _, route := range r.routes {
		if !routeAllowsOperation(route, op) {
			continue
		}

		parser, ok := r.parsers[route.ParserRef]
		if !ok {
			continue
		}

		ok2, captures, err := matchParser(parser, ctx)
		if err != nil {
			return nil, fmt.Errorf("parser %q: %w", parser.Name, err)
		}
		if !ok2 {
			continue
		}

		dests := make([]config.S3Target, 0, len(route.DestinationRefs))
		for _, ref := range route.DestinationRefs {
			if t, ok := r.targets[ref]; ok {
				dests = append(dests, t)
			}
		}
		if len(dests) == 0 {
			continue
		}

		match := Match{
			Route:        route,
			Destinations: dests,
			Captures:     captures,
		}

		if s3ops.IsRead(op) && len(dests) > 0 {
			match.EffectiveRead = selectEffectiveRead(route, dests, ctx)
		} else if len(dests) > 0 {
			match.EffectiveRead = &dests[0]
		}

		matches = append(matches, match)

		if route.OnMatch == config.MatchStop {
			break
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no matching route")
	}
	return matches, nil
}

func matchParser(p config.Parser, ctx *requestctx.Context) (bool, map[string]string, error) {
	captures := make(map[string]string)

	switch p.Kind {
	case config.ParserPathPrefix:
		// A path_prefix route matches a path either exactly equal to the
		// configured prefix or beginning with the prefix followed by a `/`.
		// This prevents `/replica` accidentally matching `/replicate/...`,
		// which otherwise surprises operators when route order is rotated.
		if ctx.RawPath == p.Prefix || strings.HasPrefix(ctx.RawPath, p.Prefix+"/") {
			return true, captures, nil
		}
		return false, nil, nil

	case config.ParserBucketExact:
		if ctx.Bucket == p.Bucket {
			return true, captures, nil
		}
		return false, nil, nil

	case config.ParserBucketRegex:
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			return false, nil, err
		}
		match := re.FindStringSubmatch(ctx.Bucket)
		if match == nil {
			return false, nil, nil
		}
		for i, name := range re.SubexpNames() {
			if i > 0 && name != "" && i < len(match) {
				captures[name] = match[i]
			}
		}
		return true, captures, nil

	case config.ParserHostSuffix:
		if strings.HasSuffix(ctx.Host, p.Suffix) {
			return true, captures, nil
		}
		return false, nil, nil
	}

	return false, nil, nil
}

func routeAllowsOperation(route config.Route, op s3ops.Operation) bool {
	for _, allowed := range route.Operations {
		if allowed == string(op) {
			return true
		}
	}
	return false
}

func selectEffectiveRead(route config.Route, dests []config.S3Target, ctx *requestctx.Context) *config.S3Target {
	if len(dests) == 0 {
		return nil
	}
	if len(dests) == 1 {
		return &dests[0]
	}

	switch route.ReadPreference {
	case config.ReadFirst:
		return &dests[0]
	case config.ReadRandom:
		return &dests[rand.Intn(len(dests))]
	case config.ReadHash:
		idx := hashIndex(ctx.Bucket, ctx.Key, len(dests))
		return &dests[idx]
	case config.ReadOrderedFailover:
		return &dests[0]
	}
	return &dests[0]
}

func hashIndex(bucket, key string, n int) int {
	if n <= 1 {
		return 0
	}
	h := sha256.New()
	h.Write([]byte(bucket))
	h.Write([]byte("/"))
	h.Write([]byte(key))
	sum := h.Sum(nil)
	var idx int
	for _, b := range sum[:4] {
		idx = (idx << 8) | int(b)
	}
	if idx < 0 {
		idx = -idx
	}
	return idx % n
}
