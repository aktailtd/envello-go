package envello

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strconv"
	"strings"
	"time"
)

// SignatureHeader is the HTTP header Envello puts the webhook signature in.
const SignatureHeader = "Envello-Signature"

// defaultWebhookTolerance is how far the signed timestamp may drift from the
// current time before VerifyWebhook rejects the request as a replay.
const defaultWebhookTolerance = 300 * time.Second

// VerifyWebhook reports whether header is a valid Envello-Signature for
// rawBody, signed with secret, within 300 seconds of now.
//
// rawBody must be the exact bytes received, not JSON you decoded and
// re-encoded. secret is the endpoint secret as shown, including its
// "whsec_" prefix.
func VerifyWebhook(secret string, rawBody []byte, header string) bool {
	return VerifyWebhookWithTolerance(secret, rawBody, header, defaultWebhookTolerance, time.Now())
}

// VerifyWebhookWithTolerance is VerifyWebhook with an explicit replay
// tolerance and clock, mainly for tests.
//
// The header looks like "t=<unix-seconds>,v1=<hex>[,v1=<hex>...]". During a
// secret rotation it carries one v1 per active secret, and a match on any of
// them passes. Malformed input returns false; it never panics.
func VerifyWebhookWithTolerance(secret string, rawBody []byte, header string, tolerance time.Duration, now time.Time) bool {
	// An unset secret (os.Getenv on a missing variable) would make every
	// header signed with an empty key verify.
	if secret == "" {
		return false
	}
	timestamp, signatures, ok := parseSignatureHeader(header)
	if !ok {
		return false
	}

	// time.Time.Sub saturates instead of overflowing, so a far-off t can't
	// wrap around into the window.
	drift := now.Sub(time.Unix(timestamp, 0))
	if drift > tolerance || drift < -tolerance {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(rawBody)
	expected := mac.Sum(nil)

	// Check every v1 with no early exit, so timing doesn't reveal which
	// signature (if any) matched.
	matched := false
	for _, signature := range signatures {
		provided, err := hex.DecodeString(signature)
		if err != nil {
			continue
		}
		if hmac.Equal(expected, provided) {
			matched = true
		}
	}
	return matched
}

// parseSignatureHeader splits "t=...,v1=...,v1=..." into the timestamp and
// the v1 signatures. ok is false unless t is a positive integer and there is
// at least one v1. Unknown keys are ignored so new schemes can be added.
func parseSignatureHeader(header string) (timestamp int64, signatures []string, ok bool) {
	hasTimestamp := false
	for _, part := range strings.Split(header, ",") {
		key, value, found := strings.Cut(part, "=")
		if !found || key == "" || value == "" {
			continue
		}
		switch key {
		case "t":
			// ParseUint rejects signs, so "+1" and "-1" fail along with
			// non-numbers.
			parsed, err := strconv.ParseUint(value, 10, 64)
			if err != nil || parsed == 0 || parsed > math.MaxInt64 {
				return 0, nil, false
			}
			timestamp = int64(parsed)
			hasTimestamp = true
		case "v1":
			signatures = append(signatures, value)
		}
	}
	if !hasTimestamp || len(signatures) == 0 {
		return 0, nil, false
	}
	return timestamp, signatures, true
}
