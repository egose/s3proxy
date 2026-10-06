package requestpayload

import (
	"errors"
	"net/http"
	"strings"
)

var ErrUnsupported = errors.New("AWS streaming payload formats are not supported")

func Validate(headers http.Header) error {
	for name, values := range headers {
		for _, value := range values {
			switch {
			case strings.EqualFold(name, "Content-Encoding"):
				for _, token := range strings.Split(value, ",") {
					if strings.EqualFold(strings.TrimSpace(token), "aws-chunked") {
						return ErrUnsupported
					}
				}
			case strings.EqualFold(name, "X-Amz-Content-Sha256"):
				if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(value)), "STREAMING-") {
					return ErrUnsupported
				}
			case strings.EqualFold(name, "X-Amz-Trailer"):
				if strings.TrimSpace(value) != "" {
					return ErrUnsupported
				}
			}
		}
	}
	return nil
}
