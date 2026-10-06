package s3signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
)

func Authorization(r *http.Request, accessKey, secret, region, signedHeaders string) string {
	date := r.Header.Get("X-Amz-Date")
	scope := date[:8] + "/" + region + "/s3/aws4_request"
	var headers strings.Builder
	for _, name := range strings.Split(signedHeaders, ";") {
		values := r.Header.Values(name)
		if name == "host" {
			values = []string{r.Host}
		}
		headers.WriteString(name + ":")
		for i, value := range values {
			if i > 0 {
				headers.WriteByte(',')
			}
			headers.WriteString(strings.Join(strings.Fields(value), " "))
		}
		headers.WriteByte('\n')
	}
	query := r.URL.Query()
	for key := range query {
		sort.Strings(query[key])
	}
	canonical := strings.Join([]string{
		r.Method, r.URL.EscapedPath(), strings.ReplaceAll(query.Encode(), "+", "%20"),
		headers.String(), signedHeaders, r.Header.Get("X-Amz-Content-Sha256"),
	}, "\n")
	sum := sha256.Sum256([]byte(canonical))
	toSign := "AWS4-HMAC-SHA256\n" + date + "\n" + scope + "\n" + hex.EncodeToString(sum[:])
	key := []byte("AWS4" + secret)
	for _, part := range []string{date[:8], region, "s3", "aws4_request", toSign} {
		mac := hmac.New(sha256.New, key)
		mac.Write([]byte(part))
		key = mac.Sum(nil)
	}
	return "AWS4-HMAC-SHA256 Credential=" + accessKey + "/" + scope + ", SignedHeaders=" + signedHeaders + ", Signature=" + hex.EncodeToString(key)
}
