package llm

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
	"aireview/internal/review"
)

const defaultUserAgent = "aireview"

type OpenAICompatibleProvider struct {
	cfg        config.Config
	httpClient *http.Client
	userAgent  string
}

type Option func(*OpenAICompatibleProvider)

func NewOpenAICompatibleProvider(cfg config.Config, opts ...Option) *OpenAICompatibleProvider {
	p := &OpenAICompatibleProvider{
		cfg:       cfg,
		userAgent: defaultUserAgent,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *OpenAICompatibleProvider) {
		if client != nil {
			p.httpClient = client
		}
	}
}

func WithUserAgent(userAgent string) Option {
	return func(p *OpenAICompatibleProvider) {
		if strings.TrimSpace(userAgent) != "" {
			p.userAgent = userAgent
		}
	}
}

func (p *OpenAICompatibleProvider) Review(ctx context.Context, req ReviewRequest) (ReviewResponse, error) {
	cfg := req.Config
	if cfg.LLM.BaseURL == "" && p.cfg.LLM.BaseURL != "" {
		cfg = p.cfg
	}
	if err := validateLLMConfig(cfg); err != nil {
		return ReviewResponse{}, err
	}
	req.Config = cfg

	apiKey := strings.TrimSpace(os.Getenv(cfg.LLM.APIKeyEnv))
	if apiKey == "" {
		return ReviewResponse{}, fmt.Errorf("LLM API key is required: set %s", cfg.LLM.APIKeyEnv)
	}

	userPrompt, err := buildPrompt(req)
	if err != nil {
		return ReviewResponse{}, err
	}
	requestBody := chatCompletionRequest{
		Model: cfg.LLM.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
	}

	var responseBody bytes.Buffer
	if err := json.NewEncoder(&responseBody).Encode(requestBody); err != nil {
		return ReviewResponse{}, fmt.Errorf("encode LLM request: %w", err)
	}

	endpoint, err := chatCompletionsEndpoint(cfg.LLM.BaseURL)
	if err != nil {
		return ReviewResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &responseBody)
	if err != nil {
		return ReviewResponse{}, fmt.Errorf("build LLM request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", p.userAgent)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return ReviewResponse{}, fmt.Errorf("call LLM provider: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return ReviewResponse{}, fmt.Errorf("read LLM response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return ReviewResponse{}, llmHTTPError(resp.StatusCode, body)
	}

	content, err := parseChatContent(body)
	if err != nil {
		return ReviewResponse{}, err
	}
	report, err := parseReviewReport(content)
	if err != nil {
		return ReviewResponse{}, err
	}

	return ReviewResponse{Report: report}, nil
}

func validateLLMConfig(cfg config.Config) error {
	if strings.TrimSpace(cfg.LLM.BaseURL) == "" {
		return errors.New("LLM base_url is required")
	}
	if strings.TrimSpace(cfg.LLM.Model) == "" {
		return errors.New("LLM model is required")
	}
	if strings.TrimSpace(cfg.LLM.APIKeyEnv) == "" {
		return errors.New("LLM api_key_env is required")
	}
	return nil
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

func parseReviewReport(content string) (review.ReviewReport, error) {
	cleaned := stripJSONFence(content)
	var report review.ReviewReport
	if err := json.Unmarshal([]byte(cleaned), &report); err != nil {
		return review.ReviewReport{}, fmt.Errorf("LLM returned non-JSON ReviewReport: %w; response excerpt: %s", err, excerpt(content, 500))
	}
	return report, nil
}

func stripJSONFence(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) >= 3 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
		return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
	}
	return content
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

func excerpt(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
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
