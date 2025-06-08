package testutil

import (
	"net/http"
)

// RoundTripFunc is an http.RoundTripper implemented by a function.
type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// NewTestClient returns an *http.Client which will return the given response
// for every request.
func NewTestClient(resp *http.Response, err error) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return resp, err
		}),
	}
}

// NewTestClientWithRequest behaves like NewTestClient but also returns the
// request received via the reqOut pointer for inspection in tests.
func NewTestClientWithRequest(resp *http.Response, err error, reqOut **http.Request) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if reqOut != nil {
				*reqOut = req
			}
			return resp, err
		}),
	}
}
