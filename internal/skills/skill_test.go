package skills

import (
	"strings"
	"testing"
)

func TestLoadBuiltins(t *testing.T) {
	r := &Registry{byName: map[string]*Skill{}}
	if err := loadBuiltins(); err != nil {
		t.Fatalf("loadBuiltins failed: %v", err)
	}
	// Verify Global registry has been populated
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	all := Global.All()
	if len(all) == 0 {
		t.Fatal("expected at least one built-in skill")
	}
	_ = r
}

func TestBuiltinSkillNames(t *testing.T) {
	_ = Load()
	expectedNames := []string{"prd", "arch", "devplan", "codedoc", "apidoc", "testplan", "refactor", "review", "debug"}
	for _, name := range expectedNames {
		s := Global.Get(name)
		if s == nil {
			t.Errorf("expected skill %q to be registered", name)
			continue
		}
		if s.Description == "" {
			t.Errorf("skill %q has empty description", name)
		}
		if s.Prompt == "" {
			t.Errorf("skill %q has empty prompt", name)
		}
	}
}

func TestSkillDetect(t *testing.T) {
	_ = Load()
	cases := []struct {
		input    string
		expected string
	}{
		{"帮我写一个产品需求文档", "prd"},
		{"请生成架构设计文档", "arch"},
		{"制定开发计划", "devplan"},
		{"给这段代码写文档注释", "codedoc"},
		{"生成API文档", "apidoc"},
		{"写测试计划", "testplan"},
		{"帮我重构这个函数", "refactor"},
		{"review一下这段代码", "review"},
		{"这个程序报错了，帮我debug", "debug"},
	}
	for _, c := range cases {
		s := Global.Detect(c.input)
		if s == nil {
			t.Errorf("Detect(%q) = nil, want %q", c.input, c.expected)
			continue
		}
		if s.Name != c.expected {
			t.Errorf("Detect(%q) = %q, want %q", c.input, s.Name, c.expected)
		}
	}
}

func TestSkillGet(t *testing.T) {
	_ = Load()
	s := Global.Get("prd")
	if s == nil {
		t.Fatal("expected to find prd skill")
	}
	if s.Name != "prd" {
		t.Errorf("unexpected name: %s", s.Name)
	}
}

func TestSkillGetCaseInsensitive(t *testing.T) {
	_ = Load()
	if Global.Get("PRD") == nil {
		t.Error("expected case-insensitive Get to work")
	}
}

func TestSkillGetPartialMatch(t *testing.T) {
	_ = Load()
	// "dev" should match "devplan"
	s := Global.Get("dev")
	if s == nil {
		t.Error("expected partial match on 'dev'")
	}
}

func TestParseSkill(t *testing.T) {
	raw := []byte(`---
name: myskill
aliases: [alias1, alias2]
description: A test skill
triggers:
  - test.*skill
  - myskill
output_file: out.md
---

# My Skill Content

This is the skill prompt.
`)
	s, err := parseSkill(raw)
	if err != nil {
		t.Fatalf("parseSkill failed: %v", err)
	}
	if s.Name != "myskill" {
		t.Errorf("unexpected name: %s", s.Name)
	}
	if s.Description != "A test skill" {
		t.Errorf("unexpected description: %s", s.Description)
	}
	if s.OutputFile != "out.md" {
		t.Errorf("unexpected output_file: %s", s.OutputFile)
	}
	if !strings.Contains(s.Prompt, "My Skill Content") {
		t.Errorf("prompt missing content: %s", s.Prompt)
	}
	if len(s.Aliases) < 2 {
		t.Errorf("expected 2 aliases, got %d: %v", len(s.Aliases), s.Aliases)
	}
}

func TestSkillMatches(t *testing.T) {
	s := &Skill{
		Name:    "test",
		Aliases: []string{"测试技能"},
	}
	if !s.Matches("这是一个测试技能") {
		t.Error("expected alias to match")
	}
	if !s.Matches("test skill") {
		t.Error("expected name to match")
	}
	if s.Matches("completely unrelated") {
		t.Error("expected no match")
	}
}

func TestDetectNoMatch(t *testing.T) {
	_ = Load()
	// Something completely unrelated
	s := Global.Detect("the weather is nice today 今天天气很好")
	if s != nil {
		t.Errorf("expected no match, got skill %q", s.Name)
	}
}
