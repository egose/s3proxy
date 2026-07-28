package s3

import (
	"context"
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

	if req.Header.Get("X-Amz-Content-Sha256") == "" {
		req.Header.Set("X-Amz-Content-Sha256", unsignedPayload)
	}

	signer := v4.NewSigner()
	return signer.SignHTTP(context.Background(), creds, req, unsignedPayload, s3ServiceName, target.Region, time.Now().UTC())
}
