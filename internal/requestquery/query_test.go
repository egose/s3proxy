package requestquery

import (
	"errors"
	"net/url"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	for _, raw := range []string{
		"list-type=2&prefix=SECRET%GG", "prefix=valid&prefix=SECRET%", "SECRET%2=value",
		"SECRET%XZ=value", "tagging=;SECRET", "tagging;SECRET=value", "x=ok&x=SECRET%GG&y=valid", // pragma: allowlist secret
	} {
		t.Run(raw, func(t *testing.T) {
			values, err := Parse(raw)
			if values != nil || !errors.Is(err, ErrMalformed) || err.Error() != "malformed request query" {
				t.Fatalf("Parse = %v, %v", values, err)
			}
			if errors.Unwrap(err) != nil {
				t.Fatal("raw parser error retained")
			}
		})
	}
	for raw, want := range map[string]url.Values{
		"": {}, "&&": {}, "bare&empty=": {"bare": {""}, "empty": {""}},
		"%70refix=a%3Bb%25%2B+c%20%E9%9B%AA&prefix=second": {"prefix": {"a;b%+ c 雪", "second"}},
		"continuation-token=opaque%2F%252F%2B%26%3D%3B":    {"continuation-token": {"opaque/%2F+&=;"}},
	} {
		t.Run(raw, func(t *testing.T) {
			got, err := Parse(raw)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("Parse = %v, %v; want %v", got, err, want)
			}
		})
	}
}
