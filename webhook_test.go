package envello

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

// Test vector shared by every Envello SDK, generated with the API's own
// signer (apps/api/src/modules/events/webhook-signing.ts).
const (
	vectorSecret    = "whsec_test_secret"
	vectorOldSecret = "whsec_old"
	vectorTimestamp = 1795000000
	vectorBody      = `{"type":"email.delivered","data":{"id":"e_1"}}`
	vectorSig       = "01fd7cd18054befb25b082d367dca85a0494a916825e50896c202a70daf7d20c"
	vectorOldSig    = "2dd8bb6eb3a2f5b581fa0fa9b72bbc243b42fad944b73591035a7c34d521c271"
)

func TestVerifyWebhookWithTolerance(t *testing.T) {
	signedAt := time.Unix(vectorTimestamp, 0)
	validHeader := "t=1795000000,v1=" + vectorSig
	rotationHeader := "t=1795000000,v1=" + vectorSig + ",v1=" + vectorOldSig

	tests := []struct {
		name   string
		secret string
		body   string
		header string
		now    time.Time
		want   bool
	}{
		{"test vector passes", vectorSecret, vectorBody, validHeader, signedAt, true},
		{"empty secret never verifies, even against an empty-key signature", "", vectorBody, "t=1795000000,v1=73457d45da4254428de731ff81d20dd03d432f84fef02aead38b2fa7c7cd5c77", signedAt, false},
		{"rotation header passes for new secret", vectorSecret, vectorBody, rotationHeader, signedAt, true},
		{"rotation header passes for old secret", vectorOldSecret, vectorBody, rotationHeader, signedAt, true},
		{"rotation header passes with old v1 first", vectorOldSecret, vectorBody, "t=1795000000,v1=" + vectorOldSig + ",v1=" + vectorSig, signedAt, true},
		{"timestamp at the edge of tolerance passes", vectorSecret, vectorBody, validHeader, signedAt.Add(300 * time.Second), true},
		{"wrong secret fails", "whsec_wrong", vectorBody, validHeader, signedAt, false},
		{"tampered body fails", vectorSecret, `{"type":"email.delivered","data":{"id":"e_2"}}`, validHeader, signedAt, false},
		{"re-serialized body fails", vectorSecret, `{"type": "email.delivered", "data": {"id": "e_1"}}`, validHeader, signedAt, false},
		{"stale timestamp fails", vectorSecret, vectorBody, validHeader, signedAt.Add(301 * time.Second), false},
		{"future timestamp beyond tolerance fails", vectorSecret, vectorBody, validHeader, signedAt.Add(-301 * time.Second), false},
		{"signature for another timestamp fails", vectorSecret, vectorBody, "t=1795000001,v1=" + vectorSig, signedAt, false},
		{"empty header fails", vectorSecret, vectorBody, "", signedAt, false},
		{"missing t fails", vectorSecret, vectorBody, "v1=" + vectorSig, signedAt, false},
		{"missing v1 fails", vectorSecret, vectorBody, "t=1795000000", signedAt, false},
		{"empty v1 fails", vectorSecret, vectorBody, "t=1795000000,v1=", signedAt, false},
		{"non-numeric t fails", vectorSecret, vectorBody, "t=abc,v1=" + vectorSig, signedAt, false},
		{"zero t fails", vectorSecret, vectorBody, "t=0,v1=" + vectorSig, time.Unix(0, 0), false},
		{"negative t fails", vectorSecret, vectorBody, "t=-1795000000,v1=" + vectorSig, signedAt, false},
		{"overflowing t fails", vectorSecret, vectorBody, "t=99999999999999999999,v1=" + vectorSig, signedAt, false},
		{"non-hex v1 fails", vectorSecret, vectorBody, "t=1795000000,v1=zz", signedAt, false},
		{"truncated v1 fails", vectorSecret, vectorBody, "t=1795000000,v1=" + vectorSig[:62], signedAt, false},
		{"odd-length v1 fails", vectorSecret, vectorBody, "t=1795000000,v1=" + vectorSig[:63], signedAt, false},
		{"other scheme only fails", vectorSecret, vectorBody, "t=1795000000,v0=" + vectorSig, signedAt, false},
		{"garbage header fails", vectorSecret, vectorBody, ",,,===,t,v1", signedAt, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyWebhookWithTolerance(tt.secret, []byte(tt.body), tt.header, 300*time.Second, tt.now)
			if got != tt.want {
				t.Errorf("VerifyWebhookWithTolerance(%q) = %v, want %v", tt.header, got, tt.want)
			}
		})
	}
}

func TestVerifyWebhookUsesCurrentTime(t *testing.T) {
	now := time.Now().Unix()
	fresh := "t=" + strconv.FormatInt(now, 10) + ",v1=" + sign(vectorSecret, now, vectorBody)
	if !VerifyWebhook(vectorSecret, []byte(vectorBody), fresh) {
		t.Error("VerifyWebhook rejected a header signed just now")
	}
	stale := now - 301
	old := "t=" + strconv.FormatInt(stale, 10) + ",v1=" + sign(vectorSecret, stale, vectorBody)
	if VerifyWebhook(vectorSecret, []byte(vectorBody), old) {
		t.Error("VerifyWebhook accepted a header signed 301s ago")
	}
}

// sign mirrors the API signer so the real-clock test can build fresh headers.
func sign(secret string, timestamp int64, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10) + "." + body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestSignatureHeaderName(t *testing.T) {
	if SignatureHeader != "Envello-Signature" {
		t.Errorf("SignatureHeader = %q, want Envello-Signature", SignatureHeader)
	}
}
