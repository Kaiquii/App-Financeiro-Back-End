package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeBotProtectionValidator struct {
	err error
}

func (v fakeBotProtectionValidator) validate(_ context.Context, _ string, _ string, _ string, _ string) error {
	return v.err
}

func TestBotProtectionAuthorize(t *testing.T) {
	validRequest := BotProtectionRequest{
		ProtectionProvider: turnstileProvider,
		TurnstileToken:     "valid-token",
	}

	tests := []struct {
		name        string
		protection  botProtection
		request     BotProtectionRequest
		wantAllowed bool
		wantDown    bool
	}{
		{
			name:        "disabled accepts legacy client",
			protection:  botProtection{mode: botProtectionDisabled},
			wantAllowed: true,
		},
		{
			name: "monitor accepts missing token",
			protection: botProtection{
				mode:       botProtectionMonitor,
				validators: map[string]botProtectionValidator{},
			},
			wantAllowed: true,
		},
		{
			name: "enforce blocks missing token",
			protection: botProtection{
				mode:       botProtectionEnforce,
				validators: map[string]botProtectionValidator{},
			},
			wantAllowed: false,
		},
		{
			name: "enforce accepts valid token",
			protection: botProtection{
				mode: botProtectionEnforce,
				validators: map[string]botProtectionValidator{
					turnstileProvider: fakeBotProtectionValidator{},
				},
			},
			request:     validRequest,
			wantAllowed: true,
		},
		{
			name: "enforce returns unavailable when verifier is down",
			protection: botProtection{
				mode: botProtectionEnforce,
				validators: map[string]botProtectionValidator{
					turnstileProvider: fakeBotProtectionValidator{err: errProtectionVerifierUnavailable},
				},
			},
			request:     validRequest,
			wantAllowed: false,
			wantDown:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision := test.protection.authorize(context.Background(), "/api/auth/request-register-code", "user@example.com", "127.0.0.1", test.request)
			if decision.allowed != test.wantAllowed {
				t.Fatalf("allowed = %v, want %v", decision.allowed, test.wantAllowed)
			}
			if decision.unavailable != test.wantDown {
				t.Fatalf("unavailable = %v, want %v", decision.unavailable, test.wantDown)
			}
		})
	}
}

func TestPlayIntegrityRequestHash(t *testing.T) {
	endpoint := "/api/auth/request-register-code"
	email := " User@Example.com "
	wantSum := sha256.Sum256([]byte("POST\n" + endpoint + "\nuser@example.com"))
	want := hex.EncodeToString(wantSum[:])

	if got := playIntegrityRequestHash(endpoint, email); got != want {
		t.Fatalf("request hash = %s, want %s", got, want)
	}
}

func TestValidatePlayIntegrityVerdict(t *testing.T) {
	payload := playIntegrityPayload{}
	payload.RequestDetails.RequestPackageName = "br.com.sobraai.app"
	payload.RequestDetails.RequestHash = "expected-hash"
	payload.AppIntegrity.AppRecognitionVerdict = "PLAY_RECOGNIZED"
	payload.DeviceIntegrity.DeviceRecognitionVerdict = []string{"MEETS_DEVICE_INTEGRITY"}
	payload.AccountDetails.AppLicensingVerdict = "LICENSED"

	if err := validatePlayIntegrityVerdict(payload, "br.com.sobraai.app", "expected-hash"); err != nil {
		t.Fatalf("valid verdict rejected: %v", err)
	}

	payload.RequestDetails.RequestHash = "different-hash"
	if err := validatePlayIntegrityVerdict(payload, "br.com.sobraai.app", "expected-hash"); err == nil {
		t.Fatal("expected mismatched request hash to be rejected")
	}

	payload.RequestDetails.RequestHash = "expected-hash"
	payload.DeviceIntegrity.DeviceRecognitionVerdict = nil
	if err := validatePlayIntegrityVerdict(payload, "br.com.sobraai.app", "expected-hash"); err == nil {
		t.Fatal("expected missing device verdict to be rejected")
	}
}

func TestRequestRegisterCodeBindsBotProtectionFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("POST", "/api/auth/request-register-code", bytes.NewBufferString(`{
		"email":"user@example.com",
		"protection_provider":"turnstile",
		"turnstile_token":"token"
	}`))
	context.Request.Header.Set("Content-Type", "application/json")

	var request RequestRegisterCodeRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		t.Fatalf("request should bind: %v", err)
	}
	if request.ProtectionProvider != turnstileProvider || request.TurnstileToken != "token" {
		t.Fatalf("unexpected protection fields: %+v", request.BotProtectionRequest)
	}
}
