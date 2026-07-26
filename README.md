# envello (Go SDK)

Minimal Go client for the [Envello](https://envello.dev) transactional
email API.

## Install

```bash
go get github.com/Bergblumestudio/envello-go
```

## Usage

```go
package main

import (
	"context"
	"log"

	envello "github.com/Bergblumestudio/envello-go"
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

## API surface

Kept intentionally small — send and validate, matching the real
`POST /emails` and `POST /emails/validate` request/response shapes of the
Envello API:

- `NewClient(apiKey string, opts ...Option) *Client`
- `(*Client) SendEmail(ctx, SendEmailRequest) (*SendEmailResponse, error)`
- `(*Client) ValidateEmail(ctx, email string) (*ValidateEmailResponse, error)`

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
