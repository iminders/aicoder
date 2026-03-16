package diff

import (
	"fmt"
	"strings"
)

// Hunk represents one changed block in a diff.
type Hunk struct {
	OldStart, OldCount int
	NewStart, NewCount int
	Lines              []string // lines prefixed with ' ', '+', or '-'
}

// Diff returns a human-readable unified diff between old and new content.
func Diff(oldContent, newContent, filename string) string {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	hunks := computeHunks(oldLines, newLines)
	if len(hunks) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- a/%s\n", filename))
	sb.WriteString(fmt.Sprintf("+++ b/%s\n", filename))
	for _, h := range hunks {
		sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldCount, h.NewStart, h.NewCount))
		for _, l := range h.Lines {
			sb.WriteString(l + "\n")
		}
	}
	return sb.String()
}

// ColorDiff returns an ANSI-coloured unified diff string.
func ColorDiff(oldContent, newContent, filename string) string {
	raw := Diff(oldContent, newContent, filename)
	if raw == "" {
		return ""
	}
	var sb strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++"):
			sb.WriteString("\033[1m" + line + "\033[0m\n")
		case strings.HasPrefix(line, "@@"):
			sb.WriteString("\033[36m" + line + "\033[0m\n")
		case strings.HasPrefix(line, "+"):
			sb.WriteString("\033[32m" + line + "\033[0m\n")
		case strings.HasPrefix(line, "-"):
			sb.WriteString("\033[31m" + line + "\033[0m\n")
		default:
			sb.WriteString(line + "\n")
		}
	}
	return sb.String()
}

// ApplyEdit replaces the first occurrence of oldStr with newStr in content.
func ApplyEdit(content, oldStr, newStr string) (string, error) {
	if !strings.Contains(content, oldStr) {
		return "", fmt.Errorf("old_string not found in file")
	}
	idx := strings.Index(content, oldStr)
	count := strings.Count(content, oldStr)
	if count > 1 {
		return "", fmt.Errorf("old_string matches %d locations — be more specific", count)
	}
	_ = idx
	return strings.Replace(content, oldStr, newStr, 1), nil
}

// ---- internal Myers-diff helpers ----

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func computeHunks(oldLines, newLines []string) []Hunk {
	type edit struct{ op rune; line string }
	edits := lcs(oldLines, newLines)

	context := 3
	var hunks []Hunk
	var cur *Hunk
	oi, ni := 1, 1

	flush := func() {
		if cur != nil {
			// trim trailing context
			for len(cur.Lines) > 0 && cur.Lines[len(cur.Lines)-1][0] == ' ' {
				cur.Lines = cur.Lines[:len(cur.Lines)-1]
				cur.OldCount--
				cur.NewCount--
			}
			if cur.OldCount > 0 || cur.NewCount > 0 {
				hunks = append(hunks, *cur)
			}
			cur = nil
		}
	}

	for _, e := range edits {
		switch e.op {
		case '=':
			if cur != nil {
				cur.Lines = append(cur.Lines, " "+e.line)
				cur.OldCount++
				cur.NewCount++
				// close hunk after enough context
				unchanged := 0
				for i := len(cur.Lines) - 1; i >= 0 && cur.Lines[i][0] == ' '; i-- {
					unchanged++
				}
				if unchanged > context*2 {
					// trim excess trailing context
					for i := 0; i < context; i++ {
						cur.Lines = cur.Lines[:len(cur.Lines)-1]
						cur.OldCount--
						cur.NewCount--
					}
					flush()
				}
			}
			oi++
			ni++
		case '-':
			if cur == nil {
				startO := max(1, oi-context)
				startN := max(1, ni-context)
				cur = &Hunk{OldStart: startO, NewStart: startN}
				// add leading context
				for i := startO; i < oi; i++ {
					if i-1 < len(oldLines) {
						cur.Lines = append(cur.Lines, " "+oldLines[i-1])
						cur.OldCount++
						cur.NewCount++
					}
				}
			}
			cur.Lines = append(cur.Lines, "-"+e.line)
			cur.OldCount++
			oi++
		case '+':
			if cur == nil {
				startO := max(1, oi-context)
				startN := max(1, ni-context)
				cur = &Hunk{OldStart: startO, NewStart: startN}
				for i := startO; i < oi; i++ {
					if i-1 < len(oldLines) {
						cur.Lines = append(cur.Lines, " "+oldLines[i-1])
						cur.OldCount++
						cur.NewCount++
					}
				}
			}
			cur.Lines = append(cur.Lines, "+"+e.line)
			cur.NewCount++
			ni++
		}
	}
	flush()
	return hunks
}

type editOp struct {
	op   rune
	line string
}

// lcs-based diff (simplified O(ND) approximation using DP).
func lcs(a, b []string) []editOp {
	n, m := len(a), len(b)
	// dp[i][j] = length of LCS of a[:i] and b[:j]
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] > dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	var ops []editOp
	i, j := n, m
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && a[i-1] == b[j-1]:
			ops = append([]editOp{{op: '=', line: a[i-1]}}, ops...)
			i--
			j--
		case j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]):
			ops = append([]editOp{{op: '+', line: b[j-1]}}, ops...)
			j--
		default:
			ops = append([]editOp{{op: '-', line: a[i-1]}}, ops...)
			i--
		}
	}
	return ops
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
