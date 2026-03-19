// Package skills implements the aicoder skill system.
// Skills are Markdown files with a YAML front-matter header that define
// specialized AI personas and structured output guidance. They are embedded
// into the binary at compile time and can also be loaded from
// ~/.aicoder/skills/ for user-defined skills.
package skills

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed builtin/*.md
var builtinFS embed.FS

// Skill represents one loaded skill definition.
type Skill struct {
	// Metadata (from YAML front-matter)
	Name        string   // canonical name, e.g. "prd"
	Aliases     []string // alternative trigger phrases
	Description string   // one-line description
	Triggers    []string // regex patterns for auto-detection
	OutputFile  string   // suggested output filename (empty = no file)

	// Content (everything after the front-matter)
	Prompt string // full skill prompt injected as additional context

	// Compiled matchers
	compiled []*regexp.Regexp
}

// Matches returns true if the input string triggers this skill.
func (s *Skill) Matches(input string) bool {
	lower := strings.ToLower(input)
	// Check explicit name / aliases
	for _, alias := range append([]string{s.Name}, s.Aliases...) {
		if strings.Contains(lower, strings.ToLower(alias)) {
			return true
		}
	}
	// Check regex triggers
	for _, re := range s.compiled {
		if re.MatchString(input) {
			return true
		}
	}
	return false
}

// ─── Registry ─────────────────────────────────────────────────────────────────

// Registry holds all loaded skills.
type Registry struct {
	skills []*Skill
	byName map[string]*Skill
}

// Global is the default registry, populated at startup.
var Global = &Registry{byName: map[string]*Skill{}}

// Load loads all built-in skills plus any user skills from ~/.aicoder/skills/.
func Load() error {
	if err := loadBuiltins(); err != nil {
		return fmt.Errorf("load built-in skills: %w", err)
	}
	loadUserSkills() // best-effort; errors are silently ignored
	return nil
}

func loadBuiltins() error {
	entries, err := builtinFS.ReadDir("builtin")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := builtinFS.ReadFile("builtin/" + e.Name())
		if err != nil {
			return err
		}
		skill, err := parseSkill(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		Global.register(skill)
	}
	return nil
}

func loadUserSkills() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".aicoder", "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // directory doesn't exist yet — normal
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		skill, err := parseSkill(data)
		if err != nil {
			continue
		}
		skill.Name = "user:" + skill.Name // namespace user skills
		Global.register(skill)
	}
}

func (r *Registry) register(s *Skill) {
	r.skills = append(r.skills, s)
	r.byName[s.Name] = s
	// Also index by aliases
	for _, alias := range s.Aliases {
		key := strings.ToLower(alias)
		if _, exists := r.byName[key]; !exists {
			r.byName[key] = s
		}
	}
}

// All returns all registered skills.
func (r *Registry) All() []*Skill { return r.skills }

// Get returns a skill by name (case-insensitive), or nil.
func (r *Registry) Get(name string) *Skill {
	if s, ok := r.byName[strings.ToLower(name)]; ok {
		return s
	}
	// Partial match on name
	lower := strings.ToLower(name)
	for _, s := range r.skills {
		if strings.HasPrefix(s.Name, lower) {
			return s
		}
	}
	return nil
}

// Detect finds the best-matching skill for a user input string.
// Returns nil if no skill matches.
func (r *Registry) Detect(input string) *Skill {
	for _, s := range r.skills {
		if s.Matches(input) {
			return s
		}
	}
	return nil
}

// ─── Parser ───────────────────────────────────────────────────────────────────

// parseSkill parses a Markdown file with YAML-like front-matter.
// Front-matter is delimited by --- lines.
func parseSkill(data []byte) (*Skill, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	s := &Skill{}
	inFrontMatter := false
	frontMatterDone := false
	var bodyLines []string
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if lineNum == 1 && line == "---" {
			inFrontMatter = true
			continue
		}
		if inFrontMatter && line == "---" {
			inFrontMatter = false
			frontMatterDone = true
			continue
		}
		if inFrontMatter {
			parseFrontMatterLine(s, line)
			continue
		}
		if frontMatterDone {
			bodyLines = append(bodyLines, line)
		}
	}

	if s.Name == "" {
		return nil, fmt.Errorf("skill is missing 'name' field in front-matter")
	}

	s.Prompt = strings.Join(bodyLines, "\n")

	// Compile regex triggers
	for _, pattern := range s.Triggers {
		re, err := regexp.Compile("(?i)" + pattern)
		if err == nil {
			s.compiled = append(s.compiled, re)
		}
	}

	return s, nil
}

// parseFrontMatterLine handles one key: value line from the front-matter.
func parseFrontMatterLine(s *Skill, line string) {
	// Handle list items: "  - value"
	trimmed := strings.TrimSpace(line)

	// Key: value
	if idx := strings.Index(trimmed, ":"); idx > 0 {
		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])

		switch key {
		case "name":
			s.Name = val
		case "description":
			s.Description = val
		case "output_file":
			s.OutputFile = strings.Trim(val, `"`)
		case "aliases", "triggers":
			// Inline list: [a, b, c]
			if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
				items := splitList(val[1 : len(val)-1])
				switch key {
				case "aliases":
					s.Aliases = items
				case "triggers":
					s.Triggers = items
				}
			}
			// Multi-line list handled by list item case below
		}
		return
	}

	// List item: "  - value"
	if strings.HasPrefix(trimmed, "- ") {
		// We need context from the previous key — track with a simple heuristic:
		// if the skill has more triggers than aliases, we're in triggers; else aliases
		val := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		val = strings.Trim(val, `"'`)
		// Assign to whichever list was last being populated
		// Heuristic: triggers tend to have regex chars
		if strings.ContainsAny(val, ".*+?()[]") || len(s.Triggers) < len(s.Aliases) {
			s.Triggers = append(s.Triggers, val)
		} else {
			s.Aliases = append(s.Aliases, val)
		}
	}
}

func splitList(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		v := strings.TrimSpace(part)
		v = strings.Trim(v, `"'`)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}
