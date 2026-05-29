package diff

import (
	"path/filepath"
	"strings"
)

const (
	FileKindSource        = "source"
	FileKindTest          = "test"
	FileKindConfig        = "config"
	FileKindDependency    = "dependency"
	FileKindDocumentation = "documentation"
	FileKindGenerated     = "generated"
)

type FileClassification struct {
	Language string
	FileKind string
}

func ClassifyFile(path string) FileClassification {
	return FileClassification{
		Language: DetectLanguage(path),
		FileKind: DetectFileKind(path),
	}
}

func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "Go"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".js", ".jsx":
		return "JavaScript"
	case ".py":
		return "Python"
	case ".java":
		return "Java"
	default:
		return ""
	}
}

func DetectFileKind(path string) string {
	normalized := normalizePath(path)
	base := strings.ToLower(filepath.Base(normalized))
	ext := strings.ToLower(filepath.Ext(base))

	if isGeneratedPath(normalized, base) {
		return FileKindGenerated
	}
	if isTestPath(normalized, base) {
		return FileKindTest
	}
	if isDependencyFile(base) {
		return FileKindDependency
	}
	if isDocumentationFile(base, ext) {
		return FileKindDocumentation
	}
	if isConfigFile(base, ext) {
		return FileKindConfig
	}
	if DetectLanguage(path) != "" {
		return FileKindSource
	}
	return ""
}

func normalizePath(path string) string {
	return strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
}

func isGeneratedPath(path, base string) bool {
	return strings.HasPrefix(path, "vendor/") ||
		strings.Contains(path, "/vendor/") ||
		strings.HasPrefix(path, "dist/") ||
		strings.Contains(path, "/dist/") ||
		strings.Contains(base, ".generated.")
}

func isTestPath(path, base string) bool {
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py") ||
		strings.HasSuffix(base, "test.java") ||
		strings.Contains(path, "/test/") ||
		strings.Contains(path, "/tests/")
}

func isDependencyFile(base string) bool {
	switch base {
	case "go.mod", "go.sum",
		"package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock",
		"requirements.txt", "pyproject.toml", "poetry.lock",
		"pom.xml", "build.gradle":
		return true
	default:
		return false
	}
}

func isDocumentationFile(base, ext string) bool {
	return ext == ".md" || ext == ".markdown" || ext == ".rst" || strings.EqualFold(base, "readme")
}

func isConfigFile(base, ext string) bool {
	if ext == ".yml" || ext == ".yaml" || ext == ".json" || ext == ".toml" {
		return true
	}
	switch base {
	case ".env", ".gitignore", ".dockerignore", "dockerfile":
		return true
	default:
		return false
	}
}
