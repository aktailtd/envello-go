package envello

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient("env_test_abc123", WithBaseURL(server.URL))
}

func TestSendEmailSuccess(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody SendEmailRequest

	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(SendEmailResponse{ID: "11111111-1111-1111-1111-111111111111", Status: StatusQueued})
	})

	resp, err := client.SendEmail(context.Background(), SendEmailRequest{
		From:    "you@yourdomain.com",
		To:      []string{"user@example.com"},
		Subject: "Hello",
		HTML:    "<p>hi</p>",
	})
	if err != nil {
		t.Fatalf("SendEmail returned error: %v", err)
	}
	if resp.ID != "11111111-1111-1111-1111-111111111111" || resp.Status != StatusQueued {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if gotAuth != "Bearer env_test_abc123" {
		t.Errorf("Authorization header = %q, want Bearer env_test_abc123", gotAuth)
	}
	if gotMethod != http.MethodPost || gotPath != "/emails" {
		t.Errorf("request = %s %s, want POST /emails", gotMethod, gotPath)
	}
	if gotBody.From != "you@yourdomain.com" || gotBody.Subject != "Hello" {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
}

func TestSendEmailRequiresHTMLOrText(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not have been sent")
	})
	_, err := client.SendEmail(context.Background(), SendEmailRequest{
		From:    "you@yourdomain.com",
		To:      []string{"user@example.com"},
		Subject: "Hello",
	})
	if err == nil {
		t.Fatal("expected an error when neither HTML nor Text is set")
	}
}

func TestSendEmailIdempotencyKeyHeader(t *testing.T) {
	var gotHeader string
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(SendEmailResponse{ID: "id", Status: StatusQueued})
	})
	_, err := client.SendEmail(context.Background(), SendEmailRequest{
		From: "a@b.com", To: []string{"c@d.com"}, Subject: "x", Text: "hi",
		IdempotencyKey: "my-key-1",
	})
	if err != nil {
		t.Fatalf("SendEmail returned error: %v", err)
	}
	if gotHeader != "my-key-1" {
		t.Errorf("Idempotency-Key header = %q, want my-key-1", gotHeader)
	}
}

func TestSendEmailRateLimited(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "rate_limited", "message": "slow down"})
	})
	_, err := client.SendEmail(context.Background(), SendEmailRequest{
		From: "a@b.com", To: []string{"c@d.com"}, Subject: "x", Text: "hi",
	})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests || apiErr.Code != "rate_limited" {
		t.Errorf("unexpected APIError: %+v", apiErr)
	}
}

func TestValidateEmail(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/emails/validate" {
			t.Errorf("path = %s, want /emails/validate", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ValidateEmailResponse{
			Email: "user@example.com", Valid: true, SyntaxValid: true, MXFound: true,
		})
	})
	resp, err := client.ValidateEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("ValidateEmail returned error: %v", err)
	}
	if !resp.Valid {
		t.Errorf("expected Valid=true, got %+v", resp)
	}
}
