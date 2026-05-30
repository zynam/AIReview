package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"aireview/internal/config"
)

type ChatModelNode struct {
	Config     config.Config
	HTTPClient *http.Client
	UserAgent  string
}

func (n ChatModelNode) Run(ctx context.Context, state State) (State, error) {
	cfg := n.Config
	if err := validateLLMConfig(cfg); err != nil {
		return State{}, err
	}
	apiKey := resolveAPIKey(cfg)
	if apiKey == "" {
		return State{}, errors.New("LLM API key is required: set llm.api_key in TOML or LLM_API_KEY")
	}

	client := n.HTTPClient
	if client == nil {
		timeout := time.Duration(cfg.Agent.TimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 180 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	requestBody := chatCompletionRequest{
		Model: cfg.LLM.Model,
		Messages: []chatMessage{
			{Role: "system", Content: state.SystemPrompt},
			{Role: "user", Content: state.UserPrompt},
		},
		Temperature: 0.2,
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(requestBody); err != nil {
		return State{}, fmt.Errorf("encode LLM request: %w", err)
	}
	endpoint, err := chatCompletionsEndpoint(cfg.LLM.BaseURL)
	if err != nil {
		return State{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return State{}, fmt.Errorf("build LLM request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(n.UserAgent) != "" {
		req.Header.Set("User-Agent", n.UserAgent)
	} else {
		req.Header.Set("User-Agent", "aireview")
	}

	started := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return State{}, fmt.Errorf("call LLM provider: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return State{}, fmt.Errorf("read LLM response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return State{}, llmHTTPError(resp.StatusCode, responseBody)
	}
	content, err := parseChatContent(responseBody)
	if err != nil {
		return State{}, err
	}

	state.RawModelOutput = content
	state.Metrics.Model = cfg.LLM.Model
	state.Metrics.DurationMillis += time.Since(started).Milliseconds()
	return state, nil
}

func validateLLMConfig(cfg config.Config) error {
	if strings.TrimSpace(cfg.LLM.BaseURL) == "" {
		return errors.New("LLM base_url is required")
	}
	if strings.TrimSpace(cfg.LLM.Model) == "" {
		return errors.New("LLM model is required")
	}
	return nil
}

func resolveAPIKey(cfg config.Config) string {
	if apiKey := strings.TrimSpace(cfg.LLM.APIKey); apiKey != "" {
		return apiKey
	}
	return strings.TrimSpace(os.Getenv("LLM_API_KEY"))
}

func chatCompletionsEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return "", fmt.Errorf("invalid LLM base_url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid LLM base_url %q", baseURL)
	}
	if strings.HasSuffix(parsed.Path, "/chat/completions") {
		return parsed.String(), nil
	}
	joined, err := url.JoinPath(parsed.Path, "chat", "completions")
	if err != nil {
		return "", fmt.Errorf("build LLM endpoint: %w", err)
	}
	parsed.Path = joined
	return parsed.String(), nil
}

func parseChatContent(body []byte) (string, error) {
	var response chatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("decode LLM response: %w", err)
	}
	if len(response.Choices) == 0 {
		return "", errors.New("LLM response did not include choices")
	}
	content := strings.TrimSpace(response.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("LLM response content is empty")
	}
	return content, nil
}

func llmHTTPError(statusCode int, body []byte) error {
	message := strings.TrimSpace(string(body))
	var apiErr apiErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil && strings.TrimSpace(apiErr.Error.Message) != "" {
		message = strings.TrimSpace(apiErr.Error.Message)
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}
	switch statusCode {
	case http.StatusTooManyRequests:
		return fmt.Errorf("LLM provider rate limited request: %s", message)
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("LLM provider authentication failed: %s", message)
	default:
		if statusCode >= 500 {
			return fmt.Errorf("LLM provider server error %d: %s", statusCode, message)
		}
		return fmt.Errorf("LLM provider returned HTTP %d: %s", statusCode, message)
	}
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

type apiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
