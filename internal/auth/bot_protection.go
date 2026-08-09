package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	botProtectionDisabled = "disabled"
	botProtectionMonitor  = "monitor"
	botProtectionEnforce  = "enforce"

	turnstileProvider     = "turnstile"
	playIntegrityProvider = "play_integrity"

	playIntegrityScope = "https://www.googleapis.com/auth/playintegrity"
)

var errProtectionVerifierUnavailable = errors.New("protection verifier unavailable")

type botProtectionValidator interface {
	validate(ctx context.Context, token string, endpoint string, email string, ipAddress string) error
}

type botProtection struct {
	mode       string
	validators map[string]botProtectionValidator
}

type protectionDecision struct {
	allowed     bool
	unavailable bool
	reason      string
}

var configuredBotProtection = botProtection{
	mode:       botProtectionDisabled,
	validators: map[string]botProtectionValidator{},
}

// ConfigureBotProtectionFromEnv configures the public e-mail endpoint checks.
// Disabled is the safe default for existing clients. Monitor validates supplied
// tokens without blocking old clients; enforce requires a valid token.
func ConfigureBotProtectionFromEnv() error {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("BOT_PROTECTION_MODE")))
	if mode == "" {
		mode = botProtectionDisabled
	}
	if mode != botProtectionDisabled && mode != botProtectionMonitor && mode != botProtectionEnforce {
		return fmt.Errorf("BOT_PROTECTION_MODE invalido: %q", mode)
	}

	protection := botProtection{
		mode:       mode,
		validators: map[string]botProtectionValidator{},
	}
	if mode == botProtectionDisabled {
		configuredBotProtection = protection
		log.Println("Protecao anti-bot desativada por configuracao")
		return nil
	}

	turnstileSecret := strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY"))
	turnstileHostname := strings.ToLower(strings.TrimSpace(os.Getenv("TURNSTILE_EXPECTED_HOSTNAME")))
	if turnstileSecret == "" || turnstileHostname == "" {
		return errors.New("TURNSTILE_SECRET_KEY e TURNSTILE_EXPECTED_HOSTNAME sao obrigatorios quando a protecao anti-bot esta ativa")
	}
	protection.validators[turnstileProvider] = &turnstileValidator{
		secret:           turnstileSecret,
		expectedHostname: turnstileHostname,
		client:           &http.Client{Timeout: 10 * time.Second},
	}

	packageName := strings.TrimSpace(os.Getenv("PLAY_INTEGRITY_PACKAGE_NAME"))
	credentialsPath := strings.TrimSpace(os.Getenv("PLAY_INTEGRITY_CREDENTIALS_FILE"))
	if credentialsPath == "" {
		credentialsPath = strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	}
	if packageName == "" || credentialsPath == "" {
		return errors.New("PLAY_INTEGRITY_PACKAGE_NAME e PLAY_INTEGRITY_CREDENTIALS_FILE sao obrigatorios quando a protecao anti-bot esta ativa")
	}

	credentialsJSON, err := os.ReadFile(credentialsPath)
	if err != nil {
		return fmt.Errorf("erro ao ler credencial do Play Integrity: %w", err)
	}
	credentials, err := google.CredentialsFromJSON(context.Background(), credentialsJSON, playIntegrityScope)
	if err != nil {
		return fmt.Errorf("credencial do Play Integrity invalida: %w", err)
	}
	client := oauth2.NewClient(context.Background(), credentials.TokenSource)
	client.Timeout = 10 * time.Second
	protection.validators[playIntegrityProvider] = &playIntegrityValidator{
		packageName: packageName,
		client:      client,
	}

	configuredBotProtection = protection
	log.Printf("Protecao anti-bot configurada em modo=%s", mode)
	return nil
}

func requireBotProtection(c *gin.Context, endpoint string, email string, ipAddress string, request BotProtectionRequest) bool {
	decision := configuredBotProtection.authorize(c.Request.Context(), endpoint, email, ipAddress, request)
	if decision.allowed {
		if configuredBotProtection.mode != botProtectionDisabled && decision.reason == "verified" {
			log.Printf("Protecao anti-bot validada endpoint=%s provider=%s email_hash=%s ip=%s", endpoint, request.ProtectionProvider, emailFingerprint(email), ipAddress)
		} else if configuredBotProtection.mode == botProtectionMonitor {
			log.Printf("Protecao anti-bot em monitoramento endpoint=%s email_hash=%s ip=%s motivo=%s", endpoint, emailFingerprint(email), ipAddress, decision.reason)
		}
		return true
	}

	log.Printf("Protecao anti-bot bloqueou solicitacao endpoint=%s email_hash=%s ip=%s motivo=%s", endpoint, emailFingerprint(email), ipAddress, decision.reason)
	if decision.unavailable {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Nao foi possivel validar a protecao de seguranca. Tente novamente."})
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Validacao anti-bot obrigatoria."})
	}
	return false
}

func (p botProtection) authorize(ctx context.Context, endpoint string, email string, ipAddress string, request BotProtectionRequest) protectionDecision {
	if p.mode == botProtectionDisabled {
		return protectionDecision{allowed: true, reason: "disabled"}
	}

	provider := strings.TrimSpace(request.ProtectionProvider)
	token := tokenForProvider(provider, request)
	if provider == "" || token == "" {
		return p.handleFailure("missing_token", false)
	}

	validator, ok := p.validators[provider]
	if !ok {
		return p.handleFailure("unsupported_provider", false)
	}

	if err := validator.validate(ctx, token, endpoint, email, ipAddress); err != nil {
		if errors.Is(err, errProtectionVerifierUnavailable) {
			return p.handleFailure("verifier_unavailable", true)
		}
		return p.handleFailure("invalid_token", false)
	}

	return protectionDecision{allowed: true, reason: "verified"}
}

func (p botProtection) handleFailure(reason string, unavailable bool) protectionDecision {
	if p.mode == botProtectionMonitor {
		return protectionDecision{allowed: true, unavailable: unavailable, reason: reason}
	}
	return protectionDecision{allowed: false, unavailable: unavailable, reason: reason}
}

func tokenForProvider(provider string, request BotProtectionRequest) string {
	switch provider {
	case turnstileProvider:
		return strings.TrimSpace(request.TurnstileToken)
	case playIntegrityProvider:
		return strings.TrimSpace(request.PlayIntegrityToken)
	default:
		return ""
	}
}

func emailFingerprint(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:8])
}

func playIntegrityRequestHash(endpoint string, email string) string {
	value := "POST\n" + endpoint + "\n" + normalizeEmail(email)
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type turnstileValidator struct {
	secret           string
	expectedHostname string
	client           *http.Client
}

func (v *turnstileValidator) validate(ctx context.Context, token string, _ string, _ string, ipAddress string) error {
	values := url.Values{
		"secret":   {v.secret},
		"response": {token},
	}
	if ipAddress != "" {
		values.Set("remoteip", ipAddress)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("criar requisicao Turnstile: %w", errProtectionVerifierUnavailable)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("chamar Turnstile: %w", errProtectionVerifierUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("Turnstile indisponivel: %w", errProtectionVerifierUnavailable)
	}

	var result turnstileResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return fmt.Errorf("resposta Turnstile invalida: %w", errProtectionVerifierUnavailable)
	}
	if !result.Success || !strings.EqualFold(result.Hostname, v.expectedHostname) {
		return errors.New("token Turnstile invalido")
	}
	return nil
}

type turnstileResponse struct {
	Success  bool   `json:"success"`
	Hostname string `json:"hostname"`
}

type playIntegrityValidator struct {
	packageName string
	client      *http.Client
}

func (v *playIntegrityValidator) validate(ctx context.Context, token string, endpoint string, email string, _ string) error {
	body, err := json.Marshal(map[string]string{"integrity_token": token})
	if err != nil {
		return errors.New("token Play Integrity invalido")
	}
	url := "https://playintegrity.googleapis.com/v1/" + url.PathEscape(v.packageName) + ":decodeIntegrityToken"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("criar requisicao Play Integrity: %w", errProtectionVerifierUnavailable)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("chamar Play Integrity: %w", errProtectionVerifierUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("Play Integrity indisponivel: %w", errProtectionVerifierUnavailable)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return errors.New("token Play Integrity invalido")
	}

	var result playIntegrityDecodeResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return fmt.Errorf("resposta Play Integrity invalida: %w", errProtectionVerifierUnavailable)
	}
	return validatePlayIntegrityVerdict(result.TokenPayloadExternal, v.packageName, playIntegrityRequestHash(endpoint, email))
}

type playIntegrityDecodeResponse struct {
	TokenPayloadExternal playIntegrityPayload `json:"tokenPayloadExternal"`
}

type playIntegrityPayload struct {
	RequestDetails struct {
		RequestPackageName string `json:"requestPackageName"`
		RequestHash        string `json:"requestHash"`
	} `json:"requestDetails"`
	AppIntegrity struct {
		AppRecognitionVerdict string `json:"appRecognitionVerdict"`
	} `json:"appIntegrity"`
	DeviceIntegrity struct {
		DeviceRecognitionVerdict []string `json:"deviceRecognitionVerdict"`
	} `json:"deviceIntegrity"`
	AccountDetails struct {
		AppLicensingVerdict string `json:"appLicensingVerdict"`
	} `json:"accountDetails"`
}

func validatePlayIntegrityVerdict(payload playIntegrityPayload, packageName string, requestHash string) error {
	if payload.RequestDetails.RequestPackageName != packageName || payload.RequestDetails.RequestHash != requestHash {
		return errors.New("solicitacao Play Integrity nao corresponde ao pedido")
	}
	if payload.AppIntegrity.AppRecognitionVerdict != "PLAY_RECOGNIZED" {
		return errors.New("aplicativo nao reconhecido pelo Google Play")
	}
	if !containsVerdict(payload.DeviceIntegrity.DeviceRecognitionVerdict, "MEETS_DEVICE_INTEGRITY") {
		return errors.New("dispositivo sem integridade exigida")
	}
	if payload.AccountDetails.AppLicensingVerdict != "LICENSED" {
		return errors.New("aplicativo sem licenca do Google Play")
	}
	return nil
}

func containsVerdict(verdicts []string, expected string) bool {
	for _, verdict := range verdicts {
		if verdict == expected {
			return true
		}
	}
	return false
}
