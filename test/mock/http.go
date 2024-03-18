// Copyright 2024 Block, Inc.

package mock

import "net/http"

// A mock of http.Transport.
type Transport struct {
	RoundTripFunc func(*http.Request) (*http.Response, error)
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.RoundTripFunc != nil {
		return t.RoundTripFunc(req)
	}
	return nil, nil
}
