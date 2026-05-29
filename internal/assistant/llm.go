package assistant

import "errors"

type textGenerator interface {
	generateText(req GeminiRequest) (string, error)
}

type fallbackGenerator struct {
	primary  textGenerator
	fallback textGenerator
}

func newFallbackGenerator(primary textGenerator, fallback textGenerator) *fallbackGenerator {
	return &fallbackGenerator{
		primary:  primary,
		fallback: fallback,
	}
}

func newAssistantGenerator() (textGenerator, error) {
	gemini, geminiErr := newGeminiClient()
	groq, groqErr := newGroqClient()

	if geminiErr != nil && groqErr != nil {
		return nil, errors.New(geminiErr.Error() + "; " + groqErr.Error())
	}
	if geminiErr != nil {
		return groq, nil
	}
	if groqErr != nil {
		return gemini, nil
	}

	return newFallbackGenerator(groq, gemini), nil
}

func (g *fallbackGenerator) generateText(req GeminiRequest) (string, error) {
	text, err := g.primary.generateText(req)
	if err == nil {
		return text, nil
	}

	if g.fallback == nil {
		return "", err
	}

	if retryAfter, ok := geminiQuotaInfo(err); ok {
		setQuotaCooldown(retryAfter)
	}

	return g.fallback.generateText(req)
}
