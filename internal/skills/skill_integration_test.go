package skills

import (
	"strings"
	"testing"
)

// TestAllBuiltinTriggersDetected verifies every built-in skill can be detected
// by at least one of its documented trigger phrases.
func TestAllBuiltinTriggersDetected(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		phrase   string
		wantName string
	}{
		// prd
		{"帮我写产品需求文档", "prd"},
		{"请生成一份PRD", "prd"},
		{"product requirement document", "prd"},
		{"需求说明书怎么写", "prd"},
		// arch
		{"帮我做架构设计", "arch"},
		{"请写一份系统设计文档", "arch"},
		{"architecture design for microservice", "arch"},
		{"技术方案怎么写", "arch"},
		// devplan
		{"制定开发计划", "devplan"},
		{"帮我做开发排期", "devplan"},
		{"sprint planning", "devplan"},
		{"任务分解一下", "devplan"},
		// codedoc
		{"给这个文件写代码注释", "codedoc"},
		{"帮我写代码文档", "codedoc"},
		{"code documentation for this module", "codedoc"},
		// apidoc
		{"生成API文档", "apidoc"},
		{"写一份接口文档", "apidoc"},
		{"openapi spec", "apidoc"},
		{"swagger documentation", "apidoc"},
		// testplan
		{"写测试计划", "testplan"},
		{"生成测试用例", "testplan"},
		{"test plan for this feature", "testplan"},
		// refactor
		{"帮我重构这段代码", "refactor"},
		{"refactor this function", "refactor"},
		{"代码优化一下", "refactor"},
		// review
		{"code review一下", "review"},
		{"帮我做代码审查", "review"},
		{"review this PR", "review"},
		// debug
		{"帮我debug这个问题", "debug"},
		{"程序报错了怎么排查", "debug"},
		{"troubleshoot this error", "debug"},
		{"这里有个bug", "debug"},
	}

	for _, c := range cases {
		t.Run(c.phrase, func(t *testing.T) {
			s := Global.Detect(c.phrase)
			if s == nil {
				t.Errorf("Detect(%q) = nil, want %q", c.phrase, c.wantName)
				return
			}
			if s.Name != c.wantName {
				t.Errorf("Detect(%q) = %q, want %q", c.phrase, s.Name, c.wantName)
			}
		})
	}
}

// TestSkillPromptsNotEmpty verifies every skill has a meaningful prompt.
func TestSkillPromptsNotEmpty(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	for _, s := range Global.All() {
		if len(strings.TrimSpace(s.Prompt)) < 100 {
			t.Errorf("skill %q prompt too short (%d chars)", s.Name, len(s.Prompt))
		}
	}
}

// TestSkillOutputFiles verifies output_file values are sane.
func TestSkillOutputFiles(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	withFile := map[string]string{
		"prd":     "prd.md",
		"arch":    "arch.md",
		"devplan": "todo.md",
		"apidoc":  "openapi.yaml",
		"testplan": "testplan.md",
	}
	for name, want := range withFile {
		s := Global.Get(name)
		if s == nil {
			t.Errorf("skill %q not found", name)
			continue
		}
		if s.OutputFile != want {
			t.Errorf("skill %q: output_file = %q, want %q", name, s.OutputFile, want)
		}
	}
	// These skills edit files in-place, no output_file
	noFile := []string{"codedoc", "refactor", "review", "debug"}
	for _, name := range noFile {
		s := Global.Get(name)
		if s == nil {
			t.Errorf("skill %q not found", name)
			continue
		}
		if s.OutputFile != "" {
			t.Errorf("skill %q should have empty output_file, got %q", name, s.OutputFile)
		}
	}
}

// TestNoFalsePositives verifies common non-skill inputs don't match any skill.
func TestNoFalsePositives(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	inputs := []string{
		"hello",
		"今天天气怎么样",
		"what time is it",
		"ls -la",
		"帮我看看这段代码",
		"解释一下这个函数",
		"git status",
		"how does TCP work",
	}
	for _, input := range inputs {
		s := Global.Detect(input)
		if s != nil {
			t.Errorf("Detect(%q) = %q, want no match (false positive)", input, s.Name)
		}
	}
}

// TestRegistryAll verifies All() returns all 9 built-in skills.
func TestRegistryAll(t *testing.T) {
	// Reset and reload
	Global = &Registry{byName: map[string]*Skill{}}
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	all := Global.All()
	if len(all) < 9 {
		t.Errorf("expected at least 9 built-in skills, got %d", len(all))
	}
}

// TestUserSkillCreation simulates user skill file parsing.
func TestUserSkillCreation(t *testing.T) {
	raw := []byte(`---
name: mycompany-pr
aliases: [PR模板, pull request]
description: 生成符合公司规范的 PR 描述
triggers:
  - PR模板
  - pull request.*描述
output_file: ""
---

# 公司 PR 规范

## PR 标题格式
[类型] 简短描述

## 必填章节
- **背景**: 为什么要做这个改动
- **方案**: 怎么做的
- **测试**: 如何验证
- **风险**: 可能影响什么
`)
	s, err := parseSkill(raw)
	if err != nil {
		t.Fatalf("parseSkill failed: %v", err)
	}
	if s.Name != "mycompany-pr" {
		t.Errorf("unexpected name: %s", s.Name)
	}
	if len(s.Aliases) < 2 {
		t.Errorf("expected 2 aliases, got %d", len(s.Aliases))
	}
	if !strings.Contains(s.Prompt, "PR 规范") {
		t.Error("prompt should contain skill content")
	}
	if s.OutputFile != "" {
		t.Errorf("expected empty output_file, got %q", s.OutputFile)
	}
}
