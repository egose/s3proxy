package s3signature

import (
	"net/http"
	"testing"
)

func TestAuthorization_AWSS3GetObjectExample(t *testing.T) {
	r, err := http.NewRequest(http.MethodGet, "https://examplebucket.s3.amazonaws.com/test.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Range", "bytes=0-9")
	r.Header.Set("X-Amz-Date", "20130524T000000Z")
	r.Header.Set("X-Amz-Content-Sha256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855") // pragma: allowlist secret
	got := Authorization(r, "example-access-key", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "us-east-1", "host;range;x-amz-content-sha256;x-amz-date")
	want := "AWS4-HMAC-SHA256 Credential=example-access-key/20130524/us-east-1/s3/aws4_request, SignedHeaders=host;range;x-amz-content-sha256;x-amz-date, Signature=f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"
	if got != want {
		t.Fatalf("Authorization = %s, want %s", got, want)
	}
}
