package auth

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/resend/resend-go/v3"
)

type authEmailMessage struct {
	to       string
	subject  string
	htmlBody string
}

type authEmailSender interface {
	send(context.Context, authEmailMessage) error
}

type unconfiguredEmailSender struct{}

func (unconfiguredEmailSender) send(context.Context, authEmailMessage) error {
	return errors.New("provedor de e-mail nao configurado")
}

var (
	emailSenderMu         sync.RWMutex
	configuredEmailSender authEmailSender = unconfiguredEmailSender{}
)

func ConfigureEmailSenderFromEnv() error {
	sender, err := newEmailSenderFromEnv()
	if err != nil {
		return err
	}

	emailSenderMu.Lock()
	configuredEmailSender = sender
	emailSenderMu.Unlock()
	return nil
}

func newEmailSenderFromEnv() (authEmailSender, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("EMAIL_PROVIDER")))
	if provider == "" {
		provider = "smtp"
	}

	switch provider {
	case "resend":
		return newResendEmailSenderFromEnv()
	case "smtp":
		return newSMTPEmailSenderFromEnv()
	default:
		return nil, fmt.Errorf("EMAIL_PROVIDER deve ser resend ou smtp")
	}
}

func sendAuthEmail(ctx context.Context, to string, subject string, htmlBody string) error {
	emailSenderMu.RLock()
	sender := configuredEmailSender
	emailSenderMu.RUnlock()

	sendContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()

	return sender.send(sendContext, authEmailMessage{
		to:       to,
		subject:  subject,
		htmlBody: htmlBody,
	})
}

type resendEmailSender struct {
	client  *resend.Client
	from    string
	replyTo string
}

func newResendEmailSenderFromEnv() (*resendEmailSender, error) {
	apiKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	if apiKey == "" {
		return nil, errors.New("RESEND_API_KEY e obrigatoria quando EMAIL_PROVIDER=resend")
	}

	from, replyTo, err := emailAddressesFromEnv("EMAIL_FROM_ADDRESS")
	if err != nil {
		return nil, err
	}

	return &resendEmailSender{
		client:  resend.NewClient(apiKey),
		from:    from,
		replyTo: replyTo,
	}, nil
}

func (s *resendEmailSender) send(ctx context.Context, message authEmailMessage) error {
	to, err := parseEmailAddress("destinatario", message.to)
	if err != nil {
		return err
	}

	_, err = s.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: message.subject,
		Html:    message.htmlBody,
		ReplyTo: s.replyTo,
	})
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}

	return nil
}

type smtpEmailSender struct {
	fromAddress string
	fromHeader  string
	replyTo     string
	host        string
	port        string
	auth        smtp.Auth
}

func newSMTPEmailSenderFromEnv() (*smtpEmailSender, error) {
	fromAddress := strings.TrimSpace(os.Getenv("SMTP_EMAIL"))
	password := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if fromAddress == "" || password == "" || host == "" || port == "" {
		return nil, errors.New("SMTP_EMAIL, SMTP_PASSWORD, SMTP_HOST e SMTP_PORT sao obrigatorias quando EMAIL_PROVIDER=smtp")
	}

	from, replyTo, err := emailAddressesFromEnv("SMTP_EMAIL")
	if err != nil {
		return nil, err
	}
	parsedFrom, err := mail.ParseAddress(fromAddress)
	if err != nil {
		return nil, fmt.Errorf("SMTP_EMAIL invalido: %w", err)
	}

	return &smtpEmailSender{
		fromAddress: parsedFrom.Address,
		fromHeader:  from,
		replyTo:     replyTo,
		host:        host,
		port:        port,
		auth:        smtp.PlainAuth("", parsedFrom.Address, password, host),
	}, nil
}

func (s *smtpEmailSender) send(ctx context.Context, message authEmailMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	to, err := parseEmailAddress("destinatario", message.to)
	if err != nil {
		return err
	}

	headers := []string{
		"From: " + s.fromHeader,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("UTF-8", message.subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=\"UTF-8\"",
	}
	if s.replyTo != "" {
		headers = append(headers, "Reply-To: "+s.replyTo)
	}

	payload := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + message.htmlBody + "\r\n")
	if err := smtp.SendMail(s.host+":"+s.port, s.auth, s.fromAddress, []string{to}, payload); err != nil {
		return fmt.Errorf("smtp: %w", err)
	}

	return nil
}

func emailAddressesFromEnv(addressVariable string) (string, string, error) {
	addressValue := strings.TrimSpace(os.Getenv(addressVariable))
	if addressValue == "" {
		return "", "", fmt.Errorf("%s e obrigatoria", addressVariable)
	}
	address, err := mail.ParseAddress(addressValue)
	if err != nil {
		return "", "", fmt.Errorf("%s invalido: %w", addressVariable, err)
	}

	name := strings.TrimSpace(os.Getenv("EMAIL_FROM_NAME"))
	if name == "" {
		name = "SobraAi"
	}
	from := (&mail.Address{Name: name, Address: address.Address}).String()

	replyToValue := strings.TrimSpace(os.Getenv("EMAIL_REPLY_TO"))
	if replyToValue == "" {
		return from, "", nil
	}
	replyTo, err := mail.ParseAddress(replyToValue)
	if err != nil {
		return "", "", fmt.Errorf("EMAIL_REPLY_TO invalido: %w", err)
	}

	return from, replyTo.Address, nil
}

func parseEmailAddress(field string, value string) (string, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%s invalido: %w", field, err)
	}
	return address.Address, nil
}
