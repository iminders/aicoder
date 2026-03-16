package diff

import (
	"strings"
	"testing"
)

func TestApplyEdit(t *testing.T) {
	content := "Hello World\nThis is a test\nGoodbye"

	result, err := ApplyEdit(content, "This is a test", "This is a modified test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "This is a modified test") {
		t.Errorf("expected modified content, got: %s", result)
	}

	_, err = ApplyEdit(content, "nonexistent string", "replacement")
	if err == nil {
		t.Error("expected error for missing old_string")
	}
}

func TestDiff(t *testing.T) {
	old := "line1\nline2\nline3\n"
	new := "line1\nline2 modified\nline3\n"
	d := Diff(old, new, "test.txt")
	if d == "" {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(d, "--- a/test.txt") {
		t.Error("missing diff header")
	}
	if !strings.Contains(d, "+line2 modified") {
		t.Error("missing added line in diff")
	}
}

func TestDiffNoChange(t *testing.T) {
	content := "unchanged content\n"
	d := Diff(content, content, "test.txt")
	if d != "" {
		t.Errorf("expected empty diff for identical files, got: %s", d)
	}
}
