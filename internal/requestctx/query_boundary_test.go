package requestctx

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/requestquery"
)

func TestFromRequest_QueryBoundary(t *testing.T) {
	for _, addressing := range []config.Addressing{{PathStyle: true}, {VirtualHosted: true}, {}} {
		r := httptest.NewRequest("DELETE", "http://proxy.local/bucket/key?x-id=DeleteObject&versionId=SECRET%GG", nil)
		ctx, err := FromRequest(r, addressing)
		if ctx != nil || !errors.Is(err, requestquery.ErrMalformed) {
			t.Fatalf("FromRequest = %v, %v", ctx, err)
		}
	}
}
