# feat/verify-webhook

## Goal
T24: give the Go SDK a webhook signature verifier that matches the API signer (`apps/api/src/modules/events/webhook-signing.ts`) and the shared T24 spec.

## What changed
- `webhook.go`: `VerifyWebhook(secret, rawBody, header) bool` (300s tolerance, `time.Now()`), `VerifyWebhookWithTolerance(secret, rawBody, header, tolerance, now) bool`, and the `SignatureHeader = "Envello-Signature"` constant. HMAC-SHA256 over `<t>.<rawBody>`, `hmac.Equal` against every `v1` with no early exit.
- `webhook_test.go`: table-driven tests on the shared vector: valid, rotation (both secrets, both orders), wrong secret, tampered and re-serialized body, t+301s, t-301s, and malformed headers (empty, missing t or v1, bad t, bad hex, truncated). Plus a real-clock test that signs a fresh header.
- `README.md`: "Verifying webhooks" section with a `net/http` handler, API surface list updated, one em dash removed.

## Decisions
- Header split on `,`, each part on the first `=` (`strings.Cut`), per the spec. The TS parser uses `split("=")`, which differs only for values containing `=`, which valid headers never have.
- `v1` is hex-decoded and compared as bytes, like the TS `Buffer.from(hex)` path, so uppercase hex verifies in both. Undecodable `v1` is skipped, not fatal.
- `t` parsed with `ParseUint`, so signs, zero and values above `MaxInt64` fail. Last `t` wins if repeated, same as TS.
- Drift checked with `now.Sub(time.Unix(t, 0))`, which saturates, so a huge `t` can't overflow back into the window.
- Default tolerance kept unexported (`defaultWebhookTolerance`); the spec only asked for the two functions and the header constant.
- No CHANGELOG in this repo, so none added. No tag.

## State
Done. Branch pushed, no PR opened (by request), no release tagged.

## Verify
```
gofmt -l .        # no output
go vet ./...
go test ./... -v  # TestVerifyWebhookWithTolerance: 24 subtests pass
```

## Gotchas
- The spec vector's `t` (1795000000) is 2026-11-18 11:06 UTC, the day this was written. Tests pin `now`, so they don't depend on the real clock; the real-clock test signs its own header.
- Customers must pass the raw body. Any middleware that decodes JSON first breaks verification.
