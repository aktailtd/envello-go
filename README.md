# envello (Go SDK)

Minimal Go client for the [Envello](https://envello.dev) transactional
email API.

## Install

```bash
go get github.com/aktailtd/envello-go
```

## Usage

```go
package main

import (
	"context"
	"log"

	envello "github.com/aktailtd/envello-go"
)

func main() {
	client := envello.NewClient("env_test_xxx")

	resp, err := client.SendEmail(context.Background(), envello.SendEmailRequest{
		From:    "you@yourdomain.com",
		To:      []string{"user@example.com"},
		Subject: "Hello from Envello",
		HTML:    "<p>Hi there</p>",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("queued: id=%s status=%s", resp.ID, resp.Status)
}
```

Override the base URL for local dev or a self-hosted deployment with
`envello.WithBaseURL(...)`:

```go
client := envello.NewClient("env_test_xxx", envello.WithBaseURL("http://localhost:3000"))
```

The base URL must use `https`. Plain `http` is only accepted for
`localhost`, `127.0.0.1` and `[::1]`; anything else makes every call return
`envello.ErrInsecureBaseURL` without sending a request.

## Verifying webhooks

Every webhook carries an `Envello-Signature` header. Check it against the
raw request body before trusting the payload:

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	if !envello.VerifyWebhook(os.Getenv("ENVELLO_WEBHOOK_SECRET"), body, r.Header.Get(envello.SignatureHeader)) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}
	// body is authentic; decode it now.
	w.WriteHeader(http.StatusNoContent)
}
```

- Pass the exact bytes you received. Decoding and re-encoding the JSON
  changes them and the check fails.
- The secret is the endpoint secret as shown, `whsec_` prefix included.
- Timestamps more than 300 seconds from now are rejected as replays. Use
  `VerifyWebhookWithTolerance(secret, body, header, tolerance, now)` to
  change the window or pin the clock in tests.
- During a secret rotation the header carries one `v1` per active secret,
  so either secret verifies.

## API surface

Kept intentionally small: send, validate and verify webhooks. The client matches the real
`POST /emails` and `POST /emails/validate` request/response shapes of the
Envello API:

- `NewClient(apiKey string, opts ...Option) *Client`
- `(*Client) SendEmail(ctx, SendEmailRequest) (*SendEmailResponse, error)`
- `(*Client) ValidateEmail(ctx, email string) (*ValidateEmailResponse, error)`
- `VerifyWebhook(secret string, rawBody []byte, header string) bool`
- `VerifyWebhookWithTolerance(secret string, rawBody []byte, header string, tolerance time.Duration, now time.Time) bool`

`SendEmailRequest.To`/`Cc`/`Bcc` are `[]string`; use a one-element slice
for a single recipient (the API also accepts a bare string, but Go has no
clean equivalent of that union type).

Non-2xx responses return a `*envello.APIError` with `StatusCode`, `Code`
(the API's machine-readable `error` field, e.g. `"rate_limited"`), and
`Message`.

## Development

```bash
go build ./...
go vet ./...
go test ./...
```
