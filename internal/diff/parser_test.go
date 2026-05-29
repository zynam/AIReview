package diff

import "testing"

func TestParseHunkHeader(t *testing.T) {
	oldStart, newStart, err := ParseHunkHeader("@@ -10,6 +20,8 @@ func example()")
	if err != nil {
		t.Fatalf("ParseHunkHeader() error = %v", err)
	}
	if oldStart != 10 || newStart != 20 {
		t.Fatalf("ParseHunkHeader() = (%d, %d), want (10, 20)", oldStart, newStart)
	}
}

func TestParsePatch(t *testing.T) {
	patch := `@@ -10,4 +10,5 @@ func example() {
 context
-old
+new
+next
 tail
@@ -30,2 +31,2 @@ func other() {
-removed
+added`

	got, err := ParsePatch(patch)
	if err != nil {
		t.Fatalf("ParsePatch() error = %v", err)
	}
	if got.Additions != 3 {
		t.Fatalf("Additions = %d, want 3", got.Additions)
	}
	if got.Deletions != 2 {
		t.Fatalf("Deletions = %d, want 2", got.Deletions)
	}
	wantAdded := []LineChange{
		{Line: 11, Content: "new"},
		{Line: 12, Content: "next"},
		{Line: 31, Content: "added"},
	}
	if len(got.AddedLines) != len(wantAdded) {
		t.Fatalf("len(AddedLines) = %d, want %d", len(got.AddedLines), len(wantAdded))
	}
	for i := range wantAdded {
		if got.AddedLines[i] != wantAdded[i] {
			t.Fatalf("AddedLines[%d] = %#v, want %#v", i, got.AddedLines[i], wantAdded[i])
		}
	}
}

func TestHasAddedLineNear(t *testing.T) {
	lines := []LineChange{
		{Line: 10, Content: "if err != nil {"},
		{Line: 12, Content: "return err"},
	}
	if !HasAddedLineNear(lines, 10, 3, func(content string) bool {
		return content == "return err"
	}) {
		t.Fatal("HasAddedLineNear() = false, want true")
	}
}
