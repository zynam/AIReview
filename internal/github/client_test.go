package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPClientGetPullRequest(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")

	var sawAuthorization bool
	var sawUserAgent bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer env-token" {
			sawAuthorization = true
		}
		if r.Header.Get("User-Agent") == defaultUserAgent {
			sawUserAgent = true
		}

		switch r.URL.Path {
		case "/repos/openai/openai-go/pulls/123":
			writeJSON(t, w, pullRequestResponse{
				Number: 123,
				Title:  "Add client",
				Body:   "Body",
				User: struct {
					Login string "json:\"login\""
				}{Login: "octocat"},
				Base: struct {
					SHA string "json:\"sha\""
				}{SHA: "base-sha"},
				Head: struct {
					SHA string "json:\"sha\""
				}{SHA: "head-sha"},
			})
		case "/repos/openai/openai-go/pulls/123/files":
			requirePagination(t, r)
			switch r.URL.Query().Get("page") {
			case "1":
				w.Header().Set("Link", fmt.Sprintf(`<%s%s?page=2&per_page=100>; rel="next", <%s%s?page=2&per_page=100>; rel="last"`, server.URL, r.URL.Path, server.URL, r.URL.Path))
				writeJSON(t, w, []fileResponse{{
					Filename:  "internal/client.go",
					Status:    "modified",
					Additions: 10,
					Deletions: 2,
					Patch:     "@@ -1 +1 @@",
				}})
			case "2":
				writeJSON(t, w, []fileResponse{{
					Filename:  "README.md",
					Status:    "added",
					Additions: 4,
				}})
			default:
				t.Fatalf("unexpected files page %q", r.URL.Query().Get("page"))
			}
		case "/repos/openai/openai-go/pulls/123/commits":
			requirePagination(t, r)
			switch r.URL.Query().Get("page") {
			case "1":
				w.Header().Set("Link", fmt.Sprintf(`<%s%s?page=2&per_page=100>; rel="next", <%s%s?page=2&per_page=100>; rel="last"`, server.URL, r.URL.Path, server.URL, r.URL.Path))
				writeJSON(t, w, []commitResponse{{
					SHA: "abcdef123456",
					Commit: struct {
						Message string "json:\"message\""
						Author  struct {
							Name string "json:\"name\""
						} "json:\"author\""
					}{
						Message: "first commit\n\nbody",
						Author: struct {
							Name string "json:\"name\""
						}{Name: "Git Author"},
					},
					Author: &struct {
						Login string "json:\"login\""
					}{Login: "gh-user"},
				}})
			case "2":
				writeJSON(t, w, []commitResponse{{
					SHA: "123456789abc",
					Commit: struct {
						Message string "json:\"message\""
						Author  struct {
							Name string "json:\"name\""
						} "json:\"author\""
					}{
						Message: "second commit",
						Author: struct {
							Name string "json:\"name\""
						}{Name: "Fallback Author"},
					},
				}})
			default:
				t.Fatalf("unexpected commits page %q", r.URL.Query().Get("page"))
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	got, err := client.GetPullRequest(context.Background(), PRRef{
		Owner:  "openai",
		Repo:   "openai-go",
		Number: 123,
	})
	if err != nil {
		t.Fatalf("GetPullRequest() error = %v", err)
	}
	if !sawAuthorization {
		t.Fatal("GetPullRequest() did not send Authorization header from GITHUB_TOKEN")
	}
	if !sawUserAgent {
		t.Fatal("GetPullRequest() did not send required User-Agent header")
	}
	if got.Title != "Add client" || got.Author != "octocat" || got.BaseSHA != "base-sha" || got.HeadSHA != "head-sha" {
		t.Fatalf("GetPullRequest() metadata = %#v", got)
	}
	if len(got.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(got.Files))
	}
	if got.Files[0].Path != "internal/client.go" || got.Files[0].Patch == "" {
		t.Fatalf("Files[0] = %#v", got.Files[0])
	}
	if len(got.Commits) != 2 {
		t.Fatalf("len(Commits) = %d, want 2", len(got.Commits))
	}
	if got.Commits[0].Author != "gh-user" {
		t.Fatalf("Commits[0].Author = %q, want gh-user", got.Commits[0].Author)
	}
	if got.Commits[1].Author != "Fallback Author" {
		t.Fatalf("Commits[1].Author = %q, want fallback author", got.Commits[1].Author)
	}
}

func TestHTTPClientErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		headers    map[string]string
		want       string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			want:       "authentication failed",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			want:       "access forbidden",
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			want:       "not found",
		},
		{
			name:       "rate limit",
			statusCode: http.StatusForbidden,
			headers: map[string]string{
				"X-RateLimit-Remaining": "0",
				"X-RateLimit-Reset":     "1770000000",
			},
			want: "rate limit exceeded",
		},
		{
			name:       "too many requests",
			statusCode: http.StatusTooManyRequests,
			want:       "rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, value := range tt.headers {
					w.Header().Set(key, value)
				}
				w.WriteHeader(tt.statusCode)
				writeJSON(t, w, apiErrorResponse{Message: "api says no"})
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken(""))
			_, err := client.GetPullRequest(context.Background(), PRRef{
				Owner:  "openai",
				Repo:   "openai-go",
				Number: 123,
			})
			if err == nil {
				t.Fatal("GetPullRequest() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("GetPullRequest() error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write JSON: %v", err)
	}
}

func requirePagination(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.URL.Query().Get("per_page"); got != "100" {
		t.Fatalf("per_page = %q, want 100", got)
	}
	if got := r.URL.Query().Get("page"); got == "" {
		t.Fatal("page query is required")
	}
}
