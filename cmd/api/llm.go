package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sharasha07/clash-bot/internal/clash"
	"google.golang.org/genai"
)

const maxToolRounds = 2

const systemInstruction = `You are a Clash Royale assistant. The tools need a player tag. Use the tag
the user gives in their message if they gave one, otherwise the tag saved on their profile: %s. If that is none,
ask the user for their game tag and do not call any tool.`

var tools = []*genai.Tool{{
	FunctionDeclarations: []*genai.FunctionDeclaration{
		{
			Name:        "get_player",
			Description: "Look up a Clash Royale player by tag.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"tag": {Type: genai.TypeString, Description: "Player tag"},
				},
			},
		},
		{
			Name:        "get_player_battle_log",
			Description: "Fetch a Clash Royale player's recent battle results",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"tag": {Type: genai.TypeString, Description: "Player tag"},
				},
			},
		},
		{
			Name:        "get_player_upcoming_chests",
			Description: "Fetch a Clash Royale player's upcoming chests.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"tag": {Type: genai.TypeString, Description: "Player tag"},
				},
			},
		},
	},
}}

//go:generate mockgen -source=llm.go -destination=../../internal/mocks/llm.go -package=mocks
type LLM interface {
	generateReply(ctx context.Context, prompt string, gameTag *string) (string, error)
}

type geminiLLM struct {
	models   *genai.Models
	model    string
	crClient clash.ClashClient
}

func newGeminiLLM(apiKey, baseURL, model string, timeout time.Duration, crClient clash.ClashClient) (*geminiLLM, error) {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: baseURL,
			Timeout: &timeout,
			RetryOptions: &genai.HTTPRetryOptions{
				Attempts: new(int32(1)),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return &geminiLLM{models: client.Models, model: model, crClient: crClient}, nil
}

func (l *geminiLLM) generateReply(ctx context.Context, prompt string, gameTag *string) (string, error) {
	contents := genai.Text(prompt)

	tag := "none"
	if gameTag != nil {
		tag = *gameTag
	}

	for range maxToolRounds {
		resp, err := l.models.GenerateContent(ctx, l.model, contents,
			&genai.GenerateContentConfig{
				Tools: tools,
				ToolConfig: &genai.ToolConfig{
					FunctionCallingConfig: &genai.FunctionCallingConfig{
						Mode: genai.FunctionCallingConfigModeAuto,
					},
				},
				SystemInstruction: &genai.Content{
					Parts: []*genai.Part{{Text: fmt.Sprintf(systemInstruction, tag)}},
				},
			})
		if err != nil {
			return "", err
		}

		calls := resp.FunctionCalls()
		if len(calls) == 0 {
			text := strings.TrimSpace(resp.Text())
			if text == "" {
				return "", errors.New("llm returned an empty response")
			}
			return text, nil
		}

		contents = append(contents, resp.Candidates[0].Content, l.execute(ctx, calls, gameTag))
	}

	return "", errors.New("llm exceeded maximum tool call rounds")
}

func (l *geminiLLM) execute(ctx context.Context, calls []*genai.FunctionCall, gameTag *string) *genai.Content {
	parts := make([]*genai.Part, 0, len(calls))

	for _, call := range calls {
		var (
			result string
			err    error
		)

		tag, _ := call.Args["tag"].(string)
		if tag == "" && gameTag != nil {
			tag = *gameTag
		}

		switch call.Name {
		case "get_player":
			result, err = l.crClient.GetPlayer(ctx, tag)
		case "get_player_battle_log":
			result, err = l.crClient.GetPlayerBattleLog(ctx, tag, 10)
		case "get_player_upcoming_chests":
			result, err = l.crClient.GetPlayersUpcomingChests(ctx, tag, 10)
		default:
			err = errors.New("unknown tool")
		}

		response := map[string]any{}

		switch {
		case err != nil:
			response["error"] = err.Error()
		default:
			var v any
			if err := json.Unmarshal([]byte(result), &v); err != nil {
				response["error"] = "unreadable tool response: " + err.Error()
			} else if obj, ok := v.(map[string]any); ok {
				response = obj
			} else {
				response["items"] = v
			}
		}

		parts = append(parts, &genai.Part{FunctionResponse: &genai.FunctionResponse{
			ID:       call.ID,
			Name:     call.Name,
			Response: response,
		}})
	}

	return &genai.Content{Role: genai.RoleUser, Parts: parts}
}
