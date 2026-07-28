package s3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/egose/s3proxy/internal/config"
)

const (
	s3ServiceName   = "s3"
	unsignedPayload = "UNSIGNED-PAYLOAD"
)

func signRequest(req *http.Request, target config.S3Target) error {
	creds := aws.Credentials{
		AccessKeyID:     target.Credentials.AccessKey,
		SecretAccessKey: target.Credentials.SecretKey,
	}

	if err := captureBodyForSigning(req); err != nil {
		return err
	}

	if req.Header.Get("X-Amz-Content-Sha256") == "" {
		req.Header.Set("X-Amz-Content-Sha256", unsignedPayload)
	}

	signer := v4.NewSigner()
	return signer.SignHTTP(context.Background(), creds, req, unsignedPayload, s3ServiceName, target.Region, time.Now().UTC())
}

func captureBodyForSigning(req *http.Request) error {
	if req.Body == nil || req.Body == http.NoBody {
		return nil
	}
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return err
		}
		req.Body = body
		return nil
	}
	return nil
}

func payloadHash(body []byte) string {
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}
