package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp/syntax"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func decodeDiagnostics(body *hclsyntax.Body, diags hcl.Diagnostics) string {
	safe := make(hcl.Diagnostics, 0, len(diags))
	for _, diag := range diags {
		copy := *diag
		copy.Expression, copy.EvalContext, copy.Extra = nil, nil, nil
		if context, sensitive := diagnosticContext(body, diag.Subject, ""); sensitive {
			copy.Detail = context + ": environment-derived expression is invalid; value withheld"
		} else if diag.Summary == "Duplicate object key" {
			copy.Detail = context + ": object keys must be unique; value withheld"
		}
		if diag.Subject == nil {
			copy.Detail = "configuration expression is invalid; value withheld"
		}
		safe = append(safe, &copy)
	}
	return safe.Error()
}

func diagnosticContext(body *hclsyntax.Body, subject *hcl.Range, context string) (string, bool) {
	if subject == nil {
		return "", false
	}
	for name, attr := range body.Attributes {
		r := attr.Range()
		if subject.Start.Byte < r.Start.Byte || subject.Start.Byte > r.End.Byte {
			continue
		}
		sensitive := false
		hclsyntax.VisitAll(attr.Expr, func(node hclsyntax.Node) hcl.Diagnostics {
			if call, ok := node.(*hclsyntax.FunctionCallExpr); ok && call.Name == "env" {
				sensitive = true
			}
			return nil
		})
		return context + ": " + name, sensitive
	}
	for _, block := range body.Blocks {
		identity := block.Type
		for _, label := range block.Labels {
			identity += fmt.Sprintf(" %q", label)
		}
		if context != "" {
			identity = context + ": " + identity
		}
		if found, sensitive := diagnosticContext(block.Body, subject, identity); found != "" {
			return found, sensitive
		}
	}
	return "", false
}

func compileError(location hcl.Range, block, name, field, cause string) error {
	return fmt.Errorf("%s: %s %q: invalid %s: %s", location.String(), block, name, field, cause)
}

func endpointCause(err error) string {
	var parseErr *url.Error
	if errors.As(err, &parseErr) {
		err = parseErr.Err
	}
	var escape url.EscapeError
	var host url.InvalidHostError
	switch {
	case errors.As(err, &escape):
		return "invalid URL escape"
	case errors.As(err, &host):
		return "invalid character in URL host"
	case strings.HasPrefix(err.Error(), "invalid port"):
		return "invalid URL port"
	default:
		return "invalid URL syntax"
	}
}

func patternCause(err error) string {
	var syntaxErr *syntax.Error
	if errors.As(err, &syntaxErr) {
		return syntaxErr.Code.String()
	}
	return "invalid regular expression syntax"
}

func templateCause(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "function ") && strings.HasSuffix(message, " not defined"):
		return "unknown template function"
	case strings.Contains(message, "unclosed action"):
		return "unclosed template action"
	case strings.HasSuffix(message, "unexpected EOF"):
		return "unexpected end of template"
	default:
		return "invalid Go template syntax"
	}
}
