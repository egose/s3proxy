//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"
)

func TestS3EscapedKeyRoundTrip(t *testing.T) {
	waitForReady(t, 60*time.Second)
	for _, route := range []string{"primary", "replica"} {
		for _, key := range []string{"space key", "percent%key", "雪.txt", "literal%2Fseparator"} {
			t.Run(route+"/"+key, func(t *testing.T) {
				path := "/" + route + "/sign-" + randHex(8) + "/" + key
				payload := []byte("escaped object bytes")
				resp, data := signedRequest(t, http.MethodPut, path, payload, nil)
				assertStatus(t, resp, data, http.StatusOK)
				resp, data = presignedRequest(t, http.MethodGet, path, nil, time.Now().UTC(), time.Minute)
				assertStatus(t, resp, data, http.StatusOK)
				if string(data) != string(payload) {
					t.Errorf("body = %q, want %q", data, payload)
				}
				resp, data = signedRequest(t, http.MethodDelete, path, nil, nil)
				assertStatus(t, resp, data, http.StatusNoContent)
			})
		}
	}
}
