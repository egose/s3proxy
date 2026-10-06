package requestquery

import (
	"errors"
	"net/url"
)

var ErrMalformed = errors.New("malformed request query")

func Parse(raw string) (url.Values, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, ErrMalformed
	}
	return values, nil
}
