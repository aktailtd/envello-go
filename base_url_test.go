package envello

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

type failingTransport struct{ t *testing.T }

func (f failingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.t.Fatalf("request to %s should not have been sent", r.URL)
	return nil, nil
}

func TestBaseURLRejected(t *testing.T) {
	for _, baseURL := range []string{
		"http://api.envello.dev",
		"http://localhost.evil.com",
		"http://localhost@evil.com",
		"ftp://localhost",
		"not a url",
		"",
		"https://",
		"http://[::1",
	} {
		t.Run(baseURL, func(t *testing.T) {
			client := NewClient("env_test_abc123",
				WithBaseURL(baseURL),
				WithHTTPClient(&http.Client{Transport: failingTransport{t}}))
			_, err := client.ValidateEmail(context.Background(), "user@example.com")
			if !errors.Is(err, ErrInsecureBaseURL) {
				t.Fatalf("err = %v, want ErrInsecureBaseURL", err)
			}
			if err.Error() != "The API URL must use https (http is only allowed for localhost)." {
				t.Fatalf("unexpected message %q", err.Error())
			}
		})
	}
}

func TestBaseURLAccepted(t *testing.T) {
	for baseURL, want := range map[string]string{
		"https://api.envello.dev":                  "https://api.envello.dev",
		"https://self-hosted.example.com/envello/": "https://self-hosted.example.com/envello",
		"http://localhost:3000":                    "http://localhost:3000",
		"HTTP://LOCALHOST:3000":                    "HTTP://LOCALHOST:3000",
		"http://127.0.0.1:3000":                    "http://127.0.0.1:3000",
		"http://[::1]:3000":                        "http://[::1]:3000",
	} {
		t.Run(baseURL, func(t *testing.T) {
			client := NewClient("env_test_abc123", WithBaseURL(baseURL))
			if client.configErr != nil {
				t.Fatalf("configErr = %v, want nil", client.configErr)
			}
			if client.baseURL != want {
				t.Fatalf("baseURL = %q, want %q", client.baseURL, want)
			}
		})
	}
}

func TestDefaultBaseURLAccepted(t *testing.T) {
	if err := NewClient("env_test_abc123").configErr; err != nil {
		t.Fatalf("configErr = %v, want nil", err)
	}
}
