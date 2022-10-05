package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbe(t *testing.T) {
	svc := ProbeService{
		client: http.DefaultClient,
	}

	testcases := []struct {
		StatusCode int
		Body       string
	}{
		{
			StatusCode: 200,
			Body:       "Hello, world!",
		},
		{
			StatusCode: 302,
			Body:       "Redirect",
		},
		{
			StatusCode: 400,
			Body:       "Request error",
		},
		{
			StatusCode: 500,
			Body:       "Server error",
		},
	}
	for _, tc := range testcases {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.StatusCode)
			fmt.Fprintln(w, tc.Body)
		}))
		defer ts.Close()

		res := svc.probeURL(context.Background(), ts.URL)
		if res.StatusCode != tc.StatusCode {
			t.Errorf("status %d != %d", res.StatusCode, tc.StatusCode)
		}
		if res.StatusCode != tc.StatusCode {
			t.Errorf("body len %d != %d", len(res.Body), len(tc.Body))
		}
	}
}
