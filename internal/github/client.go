package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"aireview/internal/review"
)

const (
	defaultBaseURL   = "https://api.github.com"
	defaultUserAgent = "aireview"
	perPage          = 100
)

// Client fetches pull request data from GitHub and maps it into domain models.
type Client interface {
	GetPullRequest(ctx context.Context, ref PRRef) (review.PullRequest, error)
}

type HTTPClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	userAgent  string
}

type Option func(*HTTPClient)

func NewClient(opts ...Option) *HTTPClient {
	c := &HTTPClient{
		baseURL:   defaultBaseURL,
		token:     os.Getenv("GITHUB_TOKEN"),
		userAgent: defaultUserAgent,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithBaseURL(baseURL string) Option {
	return func(c *HTTPClient) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithToken(token string) Option {
	return func(c *HTTPClient) {
		c.token = token
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *HTTPClient) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func WithUserAgent(userAgent string) Option {
	return func(c *HTTPClient) {
		if strings.TrimSpace(userAgent) != "" {
			c.userAgent = userAgent
		}
	}
}

func (c *HTTPClient) GetPullRequest(ctx context.Context, ref PRRef) (review.PullRequest, error) {
	if err := validateRef(ref); err != nil {
		return review.PullRequest{}, err
	}

	var prResp pullRequestResponse
	if _, err := c.getJSON(ctx, pathForPR(ref), nil, &prResp); err != nil {
		return review.PullRequest{}, err
	}

	files, err := c.getFiles(ctx, ref)
	if err != nil {
		return review.PullRequest{}, err
	}
	commits, err := c.getCommits(ctx, ref)
	if err != nil {
		return review.PullRequest{}, err
	}

	return review.PullRequest{
		Owner:   ref.Owner,
		Repo:    ref.Repo,
		Number:  prResp.Number,
		Title:   prResp.Title,
		Body:    prResp.Body,
		Author:  prResp.User.Login,
		BaseSHA: prResp.Base.SHA,
		HeadSHA: prResp.Head.SHA,
		Files:   files,
		Commits: commits,
	}, nil
}

func (c *HTTPClient) getFiles(ctx context.Context, ref PRRef) ([]review.ChangedFile, error) {
	var files []review.ChangedFile
	path := pathForPR(ref) + "/files"
	for page := 1; ; page++ {
		var pageFiles []fileResponse
		lastPage, err := c.getJSON(ctx, path, pageQuery(page), &pageFiles)
		if err != nil {
			return nil, err
		}
		for _, file := range pageFiles {
			files = append(files, review.ChangedFile{
				Path:      file.Filename,
				Status:    file.Status,
				Additions: file.Additions,
				Deletions: file.Deletions,
				Patch:     file.Patch,
			})
		}
		if page >= lastPage {
			return files, nil
		}
	}
}

func (c *HTTPClient) getCommits(ctx context.Context, ref PRRef) ([]review.Commit, error) {
	var commits []review.Commit
	path := pathForPR(ref) + "/commits"
	for page := 1; ; page++ {
		var pageCommits []commitResponse
		lastPage, err := c.getJSON(ctx, path, pageQuery(page), &pageCommits)
		if err != nil {
			return nil, err
		}
		for _, commit := range pageCommits {
			author := commit.Commit.Author.Name
			if commit.Author != nil && commit.Author.Login != "" {
				author = commit.Author.Login
			}
			commits = append(commits, review.Commit{
				SHA:     commit.SHA,
				Message: commit.Commit.Message,
				Author:  author,
			})
		}
		if page >= lastPage {
			return commits, nil
		}
	}
}

func (c *HTTPClient) getJSON(ctx context.Context, path string, query url.Values, target any) (int, error) {
	endpoint, err := c.endpoint(path, query)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("build GitHub API request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return 0, parseError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return 0, fmt.Errorf("decode GitHub API response: %w", err)
	}
	return lastPageFromLink(resp.Header.Get("Link")), nil
}

func (c *HTTPClient) endpoint(path string, query url.Values) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid GitHub API base URL: %w", err)
	}
	escapedPath, err := url.JoinPath(base.Path, path)
	if err != nil {
		return "", fmt.Errorf("build GitHub API path: %w", err)
	}
	base.Path = escapedPath
	base.RawQuery = query.Encode()
	return base.String(), nil
}

type apiErrorResponse struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url"`
}

func parseError(resp *http.Response) error {
	var body apiErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&body)
	message := strings.TrimSpace(body.Message)
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	remaining := resp.Header.Get("X-RateLimit-Remaining")
	reset := resp.Header.Get("X-RateLimit-Reset")
	if resp.StatusCode == http.StatusForbidden && remaining == "0" {
		if reset != "" {
			return fmt.Errorf("github API rate limit exceeded; reset at unix time %s: %s", reset, message)
		}
		return fmt.Errorf("github API rate limit exceeded: %s", message)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("github API rate limited: %s", message)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("github API authentication failed: %s", message)
	case http.StatusForbidden:
		return fmt.Errorf("github API access forbidden: %s", message)
	case http.StatusNotFound:
		return fmt.Errorf("github pull request or repository not found: %s", message)
	default:
		return fmt.Errorf("github API returned %s: %s", resp.Status, message)
	}
}

func validateRef(ref PRRef) error {
	if strings.TrimSpace(ref.Owner) == "" {
		return errors.New("GitHub owner is required")
	}
	if strings.TrimSpace(ref.Repo) == "" {
		return errors.New("GitHub repo is required")
	}
	if ref.Number <= 0 {
		return errors.New("GitHub pull request number must be positive")
	}
	return nil
}

func pathForPR(ref PRRef) string {
	return "repos/" + url.PathEscape(ref.Owner) + "/" + url.PathEscape(ref.Repo) + "/pulls/" + strconv.Itoa(ref.Number)
}

func pageQuery(page int) url.Values {
	return url.Values{
		"page":     []string{strconv.Itoa(page)},
		"per_page": []string{strconv.Itoa(perPage)},
	}
}

func lastPageFromLink(linkHeader string) int {
	if linkHeader == "" {
		return 1
	}
	last := 1
	for _, part := range strings.Split(linkHeader, ",") {
		sections := strings.Split(part, ";")
		if len(sections) < 2 || !strings.Contains(sections[1], `rel="last"`) {
			continue
		}
		link := strings.Trim(sections[0], " <>")
		parsed, err := url.Parse(link)
		if err != nil {
			continue
		}
		page, err := strconv.Atoi(parsed.Query().Get("page"))
		if err == nil && page > last {
			last = page
		}
	}
	return last
}
