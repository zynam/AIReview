package github

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// PRRef identifies a GitHub pull request.
type PRRef struct {
	Owner  string
	Repo   string
	Number int
}

// ParsePRURL parses a GitHub pull request URL in the form:
//
//	https://github.com/{owner}/{repo}/pull/{number}
func ParsePRURL(rawURL string) (PRRef, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return PRRef{}, fmt.Errorf("PR URL is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return PRRef{}, fmt.Errorf("invalid PR URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return PRRef{}, fmt.Errorf("invalid PR URL: expected https scheme")
	}
	if !strings.EqualFold(parsed.Hostname(), "github.com") {
		return PRRef{}, fmt.Errorf("invalid PR URL: expected github.com host")
	}

	segments := splitPath(parsed.EscapedPath())
	if len(segments) != 4 || segments[2] != "pull" {
		return PRRef{}, fmt.Errorf("invalid PR URL: expected https://github.com/{owner}/{repo}/pull/{number}")
	}

	owner, err := url.PathUnescape(segments[0])
	if err != nil {
		return PRRef{}, fmt.Errorf("invalid PR URL owner: %w", err)
	}
	repo, err := url.PathUnescape(segments[1])
	if err != nil {
		return PRRef{}, fmt.Errorf("invalid PR URL repo: %w", err)
	}
	if owner == "" || repo == "" {
		return PRRef{}, fmt.Errorf("invalid PR URL: owner and repo are required")
	}

	number, err := strconv.Atoi(segments[3])
	if err != nil || number <= 0 {
		return PRRef{}, fmt.Errorf("invalid PR URL: pull request number must be a positive integer")
	}

	return PRRef{
		Owner:  owner,
		Repo:   repo,
		Number: number,
	}, nil
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
