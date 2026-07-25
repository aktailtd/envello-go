package envello

import (
	"context"
	"errors"
	"net/http"
)

// Attachment mirrors the Envello API's attachment union: set Content for a
// base64-encoded inline attachment, or URL to have Envello fetch a remote
// file server-side. Don't set both on the same attachment.
type Attachment struct {
	Filename    string `json:"filename,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Content     string `json:"content,omitempty"` // base64-encoded
	URL         string `json:"url,omitempty"`
}

// EmailStatus mirrors the Envello API's email status enum.
type EmailStatus string

const (
	StatusQueued    EmailStatus = "queued"
	StatusScheduled EmailStatus = "scheduled"
	StatusSent      EmailStatus = "sent"
	StatusSendError EmailStatus = "send_error"
	StatusCanceled  EmailStatus = "canceled"
)

// SendEmailRequest mirrors the Envello API's send-email request shape.
// To/Cc/Bcc are always sent as JSON arrays; the API also accepts a bare
// string, but Go has no clean equivalent of that union, so use a
// one-element slice for a single recipient.
type SendEmailRequest struct {
	From    string            `json:"from"`
	To      []string          `json:"to"`
	Cc      []string          `json:"cc,omitempty"`
	Bcc     []string          `json:"bcc,omitempty"`
	ReplyTo string            `json:"reply_to,omitempty"`
	Subject string            `json:"subject"`
	HTML    string            `json:"html,omitempty"`
	Text    string            `json:"text,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	Attachments []Attachment `json:"attachments,omitempty"`

	// SendAt is an ISO 8601 timestamp with a UTC offset, e.g.
	// "2026-08-01T09:00:00+00:00". Omit to send immediately.
	SendAt string `json:"send_at,omitempty"`

	// IdempotencyKey, if set, is sent as the Idempotency-Key header so a
	// retried request with the same key + body is deduplicated server-side.
	IdempotencyKey string `json:"-"`
}

// SendEmailResponse mirrors sendEmailResponseSchema.
type SendEmailResponse struct {
	ID     string      `json:"id"`
	Status EmailStatus `json:"status"`
}

// SendEmail sends a single email via POST /emails. Returns an *APIError if
// the API responds with a non-2xx status (e.g. 429 rate limited, 422
// suppressed recipient, 400 invalid request — see APIError.Code).
func (c *Client) SendEmail(ctx context.Context, req SendEmailRequest) (*SendEmailResponse, error) {
	if req.HTML == "" && req.Text == "" {
		return nil, errors.New("envello: at least one of HTML or Text is required")
	}
	var headers map[string]string
	if req.IdempotencyKey != "" {
		headers = map[string]string{"Idempotency-Key": req.IdempotencyKey}
	}
	var resp SendEmailResponse
	if err := c.do(ctx, http.MethodPost, "/emails", req, &resp, headers); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ValidateEmailResponse mirrors validateEmailResponseSchema.
type ValidateEmailResponse struct {
	Email       string `json:"email"`
	Valid       bool   `json:"valid"`
	SyntaxValid bool   `json:"syntax_valid"`
	Disposable  bool   `json:"disposable"`
	// RoleAccount is informational only (e.g. support@, admin@) - it does
	// not affect Valid. A role address is a real, permanently deliverable
	// mailbox, unlike Disposable.
	RoleAccount bool   `json:"role_account"`
	MXFound     bool   `json:"mx_found"`
	Reason      string `json:"reason,omitempty"`
}

// ValidateEmail checks an address's syntax, disposable-domain status,
// role-account status, and MX records via POST /emails/validate. Free on
// every plan, but still requires a valid API key.
func (c *Client) ValidateEmail(ctx context.Context, email string) (*ValidateEmailResponse, error) {
	body := struct {
		Email string `json:"email"`
	}{Email: email}
	var resp ValidateEmailResponse
	if err := c.do(ctx, http.MethodPost, "/emails/validate", body, &resp, nil); err != nil {
		return nil, err
	}
	return &resp, nil
}
