package slash

import (
	"testing"
)

func TestAllCommands(t *testing.T) {
	commands := AllCommands()
	if len(commands) == 0 {
		t.Error("AllCommands() returned empty list")
	}

	// Check that we have at least the main commands
	expectedCommands := []string{"/help", "/clear", "/history", "/undo", "/diff", "/commit", "/cost", "/model", "/config", "/init", "/exit"}
	found := make(map[string]bool)

	for _, cmd := range commands {
		found[cmd.Name] = true
	}

	for _, expected := range expectedCommands {
		if !found[expected] {
			t.Errorf("Expected command %s not found in AllCommands()", expected)
		}
	}
}

func TestComplete(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "/h",
			expected: []string{"/help", "/history"},
		},
		{
			input:    "/co",
			expected: []string{"/commit", "/config", "/cost"},
		},
		{
			input:    "/exit",
			expected: []string{"/exit"},
		},
		{
			input:    "/unknown",
			expected: []string{},
		},
		{
			input:    "",
			expected: nil,
		},
		{
			input:    "not-a-command",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := Complete(tt.input)

			if tt.expected == nil {
				if matches != nil {
					t.Errorf("Complete(%q) = %v, want nil", tt.input, matches)
				}
				return
			}

			if len(matches) != len(tt.expected) {
				t.Errorf("Complete(%q) returned %d matches, want %d", tt.input, len(matches), len(tt.expected))
				return
			}

			for i, match := range matches {
				found := false
				for _, exp := range tt.expected {
					if match.Name == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Complete(%q)[%d] = %s, not in expected list %v", tt.input, i, match.Name, tt.expected)
				}
			}
		})
	}
}

func TestCompleteNames(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "/h",
			expected: []string{"/help", "/history"},
		},
		{
			input:    "/m",
			expected: []string{"/model"},
		},
		{
			input:    "/",
			expected: []string{"/help", "/clear", "/history", "/undo", "/diff", "/commit", "/cost", "/model", "/config", "/init", "/exit", "/quit", "/q"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			names := CompleteNames(tt.input)

			if len(names) < len(tt.expected) {
				t.Errorf("CompleteNames(%q) returned %d names, want at least %d", tt.input, len(names), len(tt.expected))
			}

			// Check that all expected names are present
			found := make(map[string]bool)
			for _, name := range names {
				found[name] = true
			}

			for _, exp := range tt.expected {
				if !found[exp] {
					t.Errorf("CompleteNames(%q) missing expected name %s", tt.input, exp)
				}
			}
		})
	}
}
