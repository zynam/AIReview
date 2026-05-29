package github

import "testing"

func TestParsePRURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    PRRef
		wantErr bool
	}{
		{
			name:   "normal URL",
			rawURL: "https://github.com/openai/openai-go/pull/123",
			want: PRRef{
				Owner:  "openai",
				Repo:   "openai-go",
				Number: 123,
			},
		},
		{
			name:   "trailing slash",
			rawURL: "https://github.com/openai/openai-go/pull/123/",
			want: PRRef{
				Owner:  "openai",
				Repo:   "openai-go",
				Number: 123,
			},
		},
		{
			name:    "non GitHub URL",
			rawURL:  "https://gitlab.com/openai/openai-go/pull/123",
			wantErr: true,
		},
		{
			name:    "number is not integer",
			rawURL:  "https://github.com/openai/openai-go/pull/not-a-number",
			wantErr: true,
		},
		{
			name:    "missing owner",
			rawURL:  "https://github.com/openai-go/pull/123",
			wantErr: true,
		},
		{
			name:    "missing repo",
			rawURL:  "https://github.com/openai/pull/123",
			wantErr: true,
		},
		{
			name:    "missing number",
			rawURL:  "https://github.com/openai/openai-go/pull/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePRURL(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParsePRURL() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePRURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParsePRURL() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
