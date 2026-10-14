package rewrite

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"text/template"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestctx"
)

func TestApply_TemplateExecutionDiagnosticsValueFree(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"missing capture", `REWRITELOG_PARTIAL_OUTPUT/{{ .Captures.REWRITELOG_MISSING_CAPTURE }}/{{ .Key }}`},
		{"failing literal action", `REWRITELOG_PARTIAL_OUTPUT/{{ index "REWRITELOG_ACTION_LITERAL\"雪\\value" 9999 }}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compiled, err := template.New("REWRITELOG_TEMPLATE_NAME").Option("missingkey=error").Parse(tc.source)
			if err != nil {
				t.Fatal("could not compile runtime failure fixture")
			}
			rule := config.RewriteRule{Bucket: "store", KeyTemplate: tc.source, CompiledTemplate: compiled}
			for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete} {
				t.Run(method, func(t *testing.T) {
					ctx := &requestctx.Context{Method: method, Bucket: "REWRITELOG_BUCKET", Key: "REWRITELOG_KEY"}
					result, err := New().Apply(ctx, rule, map[string]string{"tenant": "REWRITELOG_CAPTURE_VALUE"})
					if result != (Result{}) {
						t.Error("failed execution returned a partial rewrite result")
					}
					assertValueFreeTemplateDiagnostic(t, err)
				})
			}
			t.Run("HeadBucket", func(t *testing.T) {
				result, err := New().Apply(&requestctx.Context{Method: http.MethodHead, Bucket: "visible"}, rule, nil)
				if err != nil || result != (Result{Bucket: "store"}) {
					t.Error("HeadBucket did not ignore the failing object template")
				}
			})
		})
	}
}

func assertValueFreeTemplateDiagnostic(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected template execution to fail")
	}
	if err.Error() != "key_template execution failed" {
		t.Error("execution error did not retain the fixed value-free category")
	}
	wrapped := fmt.Errorf("rewrite request: %w", err)
	if errors.Is(wrapped, ErrInvalidResult) {
		t.Error("execution failure was classified as an invalid rewrite result")
	}
	var executionError template.ExecError
	if errors.As(wrapped, &executionError) || errors.Unwrap(err) != nil {
		t.Error("execution error retained a raw template cause")
	}
	for current := wrapped; current != nil; current = errors.Unwrap(current) {
		text := current.Error() + fmt.Sprintf(" %s %v %+v %#v %q", current, current, current, current, current)
		if strings.Contains(text, "REWRITELOG_") || strings.Contains(text, "雪") {
			t.Error("execution error formatting or unwrap exposed private template input")
		}
	}
}
