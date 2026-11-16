# fix/https-base-url-pinned-actions

## Goal
T18: stop the SDK sending the API key over plain http. T19: pin GitHub Actions to commit SHAs.

## What changed
- `envello.go`: `NewClient` trims trailing slashes from the base URL and validates it. Allowed: `https`, or `http` on `localhost` / `127.0.0.1` / `[::1]`. Anything else, or an unparseable URL, stores `ErrInsecureBaseURL` ("The API URL must use https (http is only allowed for localhost).") on the client, and `do()` returns it before building a request.
- `base_url_test.go`: rejected cases (remote http, `localhost.evil.com`, `localhost@evil.com`, ftp, garbage, empty, no host, bad IPv6) use a transport that fails the test if a request leaves. Accepted cases check https, http loopback, uppercase, path kept with trailing slash stripped.
- `README.md`: documents the rule.
- `.github/workflows/ci.yml`: actions pinned to SHAs.

## Decisions
- `NewClient` returns `*Client` with no error. Changing it to `(*Client, error)` breaks every caller, and panicking on config is worse. So the error is stored and returned from every call, the same way `SendEmail` already returns validation errors before sending. Exported as the sentinel `ErrInsecureBaseURL` so callers can `errors.Is` it. The message is capitalised because the spec fixes the wording.
- Same major versions kept, pinned to the latest release in each. Floating `v4` / `v5` resolved to these SHAs on 2026-11-16.

| Action | Commit | Tag |
| --- | --- | --- |
| actions/checkout | 11d5960a326750d5838078e36cf38b85af677262 | v4.4.0 |
| actions/setup-go | 40f1582b2485089dde7abd97c1529aa768e1baff | v5.6.0 |

## State
Done. No tag pushed, nothing published.

## Verify
```
go vet ./... && go test ./... -v   # 8 tests, 14 subtests, all pass
```

## Gotchas
- A misconfigured client is only noticed on the first call, not at construction.
- The workflow has no `id-token: write` and doesn't publish, so there was nothing to scope.
