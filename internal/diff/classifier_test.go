package diff

import "testing"

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		path         string
		wantLang     string
		wantFileKind string
	}{
		{path: "internal/service.go", wantLang: "Go", wantFileKind: FileKindSource},
		{path: "internal/service_test.go", wantLang: "Go", wantFileKind: FileKindTest},
		{path: "web/app.tsx", wantLang: "TypeScript", wantFileKind: FileKindSource},
		{path: "web/app.spec.ts", wantLang: "TypeScript", wantFileKind: FileKindTest},
		{path: "test_auth.py", wantLang: "Python", wantFileKind: FileKindTest},
		{path: "src/AuthTest.java", wantLang: "Java", wantFileKind: FileKindTest},
		{path: "go.mod", wantLang: "", wantFileKind: FileKindDependency},
		{path: "package.json", wantLang: "", wantFileKind: FileKindDependency},
		{path: ".github/workflows/ci.yml", wantLang: "", wantFileKind: FileKindConfig},
		{path: "docs/design.md", wantLang: "", wantFileKind: FileKindDocumentation},
		{path: "vendor/github.com/acme/pkg/client.go", wantLang: "Go", wantFileKind: FileKindGenerated},
		{path: "dist/app.js", wantLang: "JavaScript", wantFileKind: FileKindGenerated},
		{path: "internal/model.generated.go", wantLang: "Go", wantFileKind: FileKindGenerated},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ClassifyFile(tt.path)
			if got.Language != tt.wantLang {
				t.Fatalf("Language = %q, want %q", got.Language, tt.wantLang)
			}
			if got.FileKind != tt.wantFileKind {
				t.Fatalf("FileKind = %q, want %q", got.FileKind, tt.wantFileKind)
			}
		})
	}
}
