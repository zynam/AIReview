package diff

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

type LineChange struct {
	Line    int
	Content string
}

type PatchSummary struct {
	Additions    int
	Deletions    int
	AddedLines   []LineChange
	DeletedLines []LineChange
}

func ParsePatch(patch string) (PatchSummary, error) {
	var summary PatchSummary
	var oldLine, newLine int
	inHunk := false

	for _, line := range splitPatchLines(patch) {
		if strings.HasPrefix(line, "@@") {
			oldStart, newStart, err := ParseHunkHeader(line)
			if err != nil {
				return PatchSummary{}, err
			}
			oldLine = oldStart
			newLine = newStart
			inHunk = true
			continue
		}
		if !inHunk || strings.HasPrefix(line, `\ No newline at end of file`) {
			continue
		}

		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			summary.Additions++
			summary.AddedLines = append(summary.AddedLines, LineChange{
				Line:    newLine,
				Content: strings.TrimPrefix(line, "+"),
			})
			newLine++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			summary.Deletions++
			summary.DeletedLines = append(summary.DeletedLines, LineChange{
				Line:    oldLine,
				Content: strings.TrimPrefix(line, "-"),
			})
			oldLine++
		default:
			oldLine++
			newLine++
		}
	}

	return summary, nil
}

func ParseHunkHeader(header string) (oldStart int, newStart int, err error) {
	matches := hunkHeaderPattern.FindStringSubmatch(header)
	if len(matches) != 3 {
		return 0, 0, fmt.Errorf("invalid hunk header %q", header)
	}
	oldStart, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid old hunk start %q: %w", matches[1], err)
	}
	newStart, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid new hunk start %q: %w", matches[2], err)
	}
	return oldStart, newStart, nil
}

func splitPatchLines(patch string) []string {
	patch = strings.ReplaceAll(patch, "\r\n", "\n")
	patch = strings.TrimSuffix(patch, "\n")
	if patch == "" {
		return nil
	}
	return strings.Split(patch, "\n")
}
