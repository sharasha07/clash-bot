package main

import (
	"context"
	"time"

	"google.golang.org/genai"
)

//go:generate mockgen -source=llm.go -destination=../../internal/mocks/llm.go -package=mocks
type LLM interface {
	GenerateReply(ctx context.Context, prompt string) (string, error)
}

type geminiLLM struct {
	models *genai.Models
	model  string
}

func newGeminiLLM(apiKey, baseURL, model string, timeout time.Duration) (*geminiLLM, error) {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: baseURL,
			Timeout: &timeout,
			RetryOptions: &genai.HTTPRetryOptions{
				Attempts: new(int32(2)),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return &geminiLLM{models: client.Models, model: model}, nil
}

func (l *geminiLLM) GenerateReply(ctx context.Context, prompt string) (string, error) {
	resp, err := l.models.GenerateContent(ctx, l.model, genai.Text(prompt), nil)
	if err != nil {
		return "", err
	}

	return resp.Text(), nil
}
