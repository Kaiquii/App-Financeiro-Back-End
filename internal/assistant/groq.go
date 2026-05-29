package assistant

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultGroqModel = "llama-3.1-8b-instant"

type groqClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type groqResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func newGroqClient() (*groqClient, error) {
	apiKey := strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
	if apiKey == "" {
		return nil, errors.New("GROQ_API_KEY nao configurada")
	}

	model := strings.TrimSpace(os.Getenv("GROQ_MODEL"))
	if model == "" {
		model = defaultGroqModel
	}

	return &groqClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (g *groqClient) generateText(req GeminiRequest) (string, error) {
	start := time.Now()
	stage := req.DebugStage
	if stage == "" {
		stage = "unknown"
	}

	messages := geminiRequestToGroqMessages(req)
	body, err := json.Marshal(groqRequest{
		Model:       g.model,
		Messages:    messages,
		Temperature: temperatureFromConfig(req.GenerationConfig),
	})
	if err != nil {
		return "", err
	}

	log.Printf(
		"Groq START stage=%s model=%s prompt_chars=%d messages=%d",
		stage,
		g.model,
		geminiPromptSize(req),
		len(messages),
	)

	httpReq, err := http.NewRequest(http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("Groq ERROR stage=%s model=%s duration=%s transport_error=%v", stage, g.model, time.Since(start), err)
		return "", err
	}
	defer resp.Body.Close()

	var result groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Groq ERROR stage=%s model=%s duration=%s status=%d decode_error=%v", stage, g.model, time.Since(start), resp.StatusCode, err)
		return "", err
	}

	if resp.StatusCode >= 400 {
		message := fmt.Sprintf("groq retornou status %d", resp.StatusCode)
		if result.Error != nil && result.Error.Message != "" {
			message = result.Error.Message
		}
		log.Printf("Groq ERROR stage=%s model=%s duration=%s status=%d error=%s", stage, g.model, time.Since(start), resp.StatusCode, compactLogText(message, 220))
		return "", errors.New(message)
	}

	if len(result.Choices) == 0 {
		return "", errors.New("groq nao retornou resposta")
	}

	text := strings.TrimSpace(result.Choices[0].Message.Content)
	log.Printf("Groq OK stage=%s model=%s duration=%s status=%d chars=%d", stage, g.model, time.Since(start), resp.StatusCode, len(text))
	return text, nil
}

func geminiRequestToGroqMessages(req GeminiRequest) []groqMessage {
	messages := make([]groqMessage, 0, len(req.Contents)+1)
	if req.SystemInstruction != nil {
		messages = append(messages, groqMessage{
			Role:    "system",
			Content: geminiContentText(*req.SystemInstruction),
		})
	}

	for _, content := range req.Contents {
		role := "user"
		if content.Role == "model" {
			role = "assistant"
		}
		messages = append(messages, groqMessage{
			Role:    role,
			Content: geminiContentText(content),
		})
	}
	return messages
}

func geminiContentText(content GeminiContent) string {
	parts := make([]string, 0, len(content.Parts))
	for _, part := range content.Parts {
		if strings.TrimSpace(part.Text) != "" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func temperatureFromConfig(config map[string]any) float64 {
	if config == nil {
		return 0.2
	}
	value, ok := config["temperature"]
	if !ok {
		return 0.2
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	default:
		return 0.2
	}
}
