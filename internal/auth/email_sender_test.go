package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/resend/resend-go/v3"
)

func TestNewEmailSenderFromEnvRejectsInvalidProvider(t *testing.T) {
	t.Setenv("EMAIL_PROVIDER", "invalid")

	if _, err := newEmailSenderFromEnv(); err == nil {
		t.Fatal("expected invalid provider error")
	}
}

func TestNewResendEmailSenderFromEnvRequiresAPIKey(t *testing.T) {
	t.Setenv("RESEND_API_KEY", "")
	t.Setenv("EMAIL_FROM_ADDRESS", "contato@sobraai.app.br")

	if _, err := newResendEmailSenderFromEnv(); err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestNewResendEmailSenderFromEnvBuildsAddresses(t *testing.T) {
	t.Setenv("RESEND_API_KEY", "re_test")
	t.Setenv("EMAIL_FROM_ADDRESS", "contato@sobraai.app.br")
	t.Setenv("EMAIL_FROM_NAME", "SobraAi")
	t.Setenv("EMAIL_REPLY_TO", "suporte@example.com")

	sender, err := newResendEmailSenderFromEnv()
	if err != nil {
		t.Fatalf("newResendEmailSenderFromEnv() error = %v", err)
	}
	if sender.from != `"SobraAi" <contato@sobraai.app.br>` {
		t.Fatalf("from = %q", sender.from)
	}
	if sender.replyTo != "suporte@example.com" {
		t.Fatalf("replyTo = %q", sender.replyTo)
	}
}

func TestResendEmailSenderSendsExpectedPayload(t *testing.T) {
	var received struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
		ReplyTo string   `json:"reply_to"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/emails" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer re_test" {
			t.Errorf("Authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"email_test"}`))
	}))
	defer server.Close()

	client := resend.NewCustomClient(server.Client(), "re_test")
	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	client.BaseURL = baseURL

	sender := &resendEmailSender{
		client:  client,
		from:    "SobraAi <contato@sobraai.app.br>",
		replyTo: "suporte@example.com",
	}
	err = sender.send(context.Background(), authEmailMessage{
		to:       "usuario@example.com",
		subject:  "Código para criar sua conta",
		htmlBody: "<strong>123456</strong>",
	})
	if err != nil {
		t.Fatalf("send() error = %v", err)
	}

	if received.From != sender.from || len(received.To) != 1 || received.To[0] != "usuario@example.com" {
		t.Fatalf("unexpected sender or recipient: %+v", received)
	}
	if received.Subject != "Código para criar sua conta" || received.HTML != "<strong>123456</strong>" {
		t.Fatalf("unexpected content: %+v", received)
	}
	if received.ReplyTo != sender.replyTo {
		t.Fatalf("reply_to = %q", received.ReplyTo)
	}
}
