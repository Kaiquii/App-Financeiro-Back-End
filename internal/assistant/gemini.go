package assistant

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultGeminiModel = "gemini-2.5-flash"

var geminiQuotaState = struct {
	sync.Mutex
	blockedUntil time.Time
}{}

type geminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type GeminiError struct {
	Message           string
	StatusCode        int
	QuotaExceeded     bool
	RetryAfterSeconds int
}

func (e GeminiError) Error() string {
	return e.Message
}

func newGeminiClient() (*geminiClient, error) {
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY nao configurada")
	}

	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = defaultGeminiModel
	}

	return &geminiClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (g *geminiClient) generate(req GeminiRequest) (GeminiResponse, error) {
	var result GeminiResponse
	start := time.Now()
	stage := req.DebugStage
	if stage == "" {
		stage = "unknown"
	}

	if retryAfter, blocked := currentQuotaCooldown(); blocked {
		log.Printf("Gemini SKIP stage=%s model=%s reason=quota_cooldown retry_after=%ds", stage, g.model, retryAfter)
		return result, GeminiError{
			Message:           "gemini quota cooldown ativo",
			StatusCode:        http.StatusTooManyRequests,
			QuotaExceeded:     true,
			RetryAfterSeconds: retryAfter,
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return result, err
	}

	log.Printf(
		"Gemini START stage=%s model=%s prompt_chars=%d contents=%d tools=%d",
		stage,
		g.model,
		geminiPromptSize(req),
		len(req.Contents),
		len(req.Tools),
	)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("Gemini ERROR stage=%s model=%s duration=%s transport_error=%v", stage, g.model, time.Since(start), err)
		return result, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Gemini ERROR stage=%s model=%s duration=%s status=%d decode_error=%v", stage, g.model, time.Since(start), resp.StatusCode, err)
		return result, err
	}

	if resp.StatusCode >= 400 {
		if result.Error != nil && result.Error.Message != "" {
			err := buildGeminiError(result.Error.Message, resp.StatusCode)
			if retryAfter, ok := geminiQuotaInfo(err); ok {
				setQuotaCooldown(retryAfter)
				log.Printf("Gemini QUOTA stage=%s model=%s duration=%s status=%d retry_after=%ds error=%s", stage, g.model, time.Since(start), resp.StatusCode, retryAfter, compactLogText(result.Error.Message, 220))
			} else {
				log.Printf("Gemini ERROR stage=%s model=%s duration=%s status=%d error=%s", stage, g.model, time.Since(start), resp.StatusCode, compactLogText(result.Error.Message, 220))
			}
			return result, err
		}
		log.Printf("Gemini ERROR stage=%s model=%s duration=%s status=%d", stage, g.model, time.Since(start), resp.StatusCode)
		return result, fmt.Errorf("gemini retornou status %d", resp.StatusCode)
	}

	log.Printf("Gemini OK stage=%s model=%s duration=%s status=%d candidates=%d", stage, g.model, time.Since(start), resp.StatusCode, len(result.Candidates))
	return result, nil
}

func (g *geminiClient) generateText(req GeminiRequest) (string, error) {
	resp, err := g.generate(req)
	if err != nil {
		return "", err
	}
	return extractText(resp), nil
}

func currentQuotaCooldown() (int, bool) {
	geminiQuotaState.Lock()
	defer geminiQuotaState.Unlock()

	if geminiQuotaState.blockedUntil.IsZero() || time.Now().After(geminiQuotaState.blockedUntil) {
		return 0, false
	}

	return int(time.Until(geminiQuotaState.blockedUntil).Seconds()) + 1, true
}

func setQuotaCooldown(retryAfterSeconds int) {
	if retryAfterSeconds <= 0 {
		return
	}

	geminiQuotaState.Lock()
	defer geminiQuotaState.Unlock()

	blockedUntil := time.Now().Add(time.Duration(retryAfterSeconds) * time.Second)
	if blockedUntil.After(geminiQuotaState.blockedUntil) {
		geminiQuotaState.blockedUntil = blockedUntil
	}
}

func buildGeminiError(message string, statusCode int) error {
	lower := strings.ToLower(message)
	err := GeminiError{
		Message:           message,
		StatusCode:        statusCode,
		QuotaExceeded:     strings.Contains(lower, "quota") || strings.Contains(lower, "rate-limit") || strings.Contains(lower, "rate limit"),
		RetryAfterSeconds: extractRetryAfterSeconds(message),
	}
	return err
}

func geminiPromptSize(req GeminiRequest) int {
	total := 0
	if req.SystemInstruction != nil {
		total += geminiContentSize(*req.SystemInstruction)
	}
	for _, content := range req.Contents {
		total += geminiContentSize(content)
	}
	return total
}

func geminiContentSize(content GeminiContent) int {
	total := 0
	for _, part := range content.Parts {
		total += len(part.Text)
	}
	return total
}

func compactLogText(text string, max int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= max {
		return text
	}
	return text[:max] + "..."
}

func extractRetryAfterSeconds(message string) int {
	re := regexp.MustCompile(`(?i)retry in ([0-9]+(?:\.[0-9]+)?)s`)
	matches := re.FindStringSubmatch(message)
	if len(matches) < 2 {
		return 0
	}

	seconds, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}
	return int(seconds) + 1
}

func geminiQuotaInfo(err error) (int, bool) {
	var geminiErr GeminiError
	if errors.As(err, &geminiErr) && geminiErr.QuotaExceeded {
		return geminiErr.RetryAfterSeconds, true
	}
	return 0, false
}

func extractText(resp GeminiResponse) string {
	if len(resp.Candidates) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			builder.WriteString(part.Text)
		}
	}
	return strings.TrimSpace(builder.String())
}

func extractFunctionCall(resp GeminiResponse) *FunctionCall {
	if len(resp.Candidates) == 0 {
		return nil
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if part.FunctionCall != nil {
			return part.FunctionCall
		}
	}
	return nil
}
