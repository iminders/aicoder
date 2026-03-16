package tools

import (
	"testing"
)

func TestClassifyCommandRisk(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		wantRisk RiskLevel
	}{
		// Critical/High risk commands
		{
			name:     "rm -rf root",
			command:  "rm -rf /",
			wantRisk: RiskHigh,
		},
		{
			name:     "mkfs dangerous",
			command:  "mkfs /dev/sda1",
			wantRisk: RiskHigh,
		},
		{
			name:     "fork bomb",
			command:  ":(){:|:&};:",
			wantRisk: RiskHigh,
		},
		{
			name:     "sudo command",
			command:  "sudo apt install package",
			wantRisk: RiskHigh,
		},
		{
			name:     "rm -rf directory",
			command:  "rm -rf /tmp/test",
			wantRisk: RiskHigh,
		},
		{
			name:     "shutdown",
			command:  "shutdown -h now",
			wantRisk: RiskHigh,
		},

		// Medium risk commands
		{
			name:     "rm single file",
			command:  "rm test.txt",
			wantRisk: RiskMedium,
		},
		{
			name:     "mv file",
			command:  "mv old.txt new.txt",
			wantRisk: RiskMedium,
		},
		{
			name:     "wget download",
			command:  "wget https://example.com/file.tar.gz",
			wantRisk: RiskMedium,
		},
		{
			name:     "npm install",
			command:  "npm install package",
			wantRisk: RiskMedium,
		},
		{
			name:     "git push",
			command:  "git push origin main",
			wantRisk: RiskMedium,
		},
		{
			name:     "chmod file",
			command:  "chmod 644 file.txt",
			wantRisk: RiskMedium,
		},

		// Low risk commands
		{
			name:     "ls directory",
			command:  "ls -la",
			wantRisk: RiskLow,
		},
		{
			name:     "cat file",
			command:  "cat README.md",
			wantRisk: RiskLow,
		},
		{
			name:     "git status",
			command:  "git status",
			wantRisk: RiskLow,
		},
		{
			name:     "git diff",
			command:  "git diff HEAD",
			wantRisk: RiskLow,
		},
		{
			name:     "grep search",
			command:  "grep -r 'pattern' .",
			wantRisk: RiskLow,
		},
		{
			name:     "ps list",
			command:  "ps aux",
			wantRisk: RiskLow,
		},
		{
			name:     "echo text",
			command:  "echo 'hello world'",
			wantRisk: RiskLow,
		},

		// Unknown commands (default to medium)
		{
			name:     "custom script",
			command:  "./my-script.sh",
			wantRisk: RiskMedium,
		},
		{
			name:     "unknown command",
			command:  "foobar --option",
			wantRisk: RiskMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommandRisk(tt.command)
			if got != tt.wantRisk {
				t.Errorf("ClassifyCommandRisk(%q) = %v, want %v", tt.command, got, tt.wantRisk)
			}
		})
	}
}

func TestClassifyToolRisk(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		params   map[string]interface{}
		wantRisk RiskLevel
	}{
		// Read-only tools
		{
			name:     "read_file",
			toolName: "read_file",
			params:   map[string]interface{}{"path": "test.txt"},
			wantRisk: RiskLow,
		},
		{
			name:     "list_dir",
			toolName: "list_dir",
			params:   map[string]interface{}{"path": "."},
			wantRisk: RiskLow,
		},
		{
			name:     "grep_search",
			toolName: "grep_search",
			params:   map[string]interface{}{"pattern": "test"},
			wantRisk: RiskLow,
		},

		// Write operations
		{
			name:     "write_file normal",
			toolName: "write_file",
			params:   map[string]interface{}{"path": "test.txt"},
			wantRisk: RiskMedium,
		},
		{
			name:     "write_file dangerous path",
			toolName: "write_file",
			params:   map[string]interface{}{"path": "/etc/passwd"},
			wantRisk: RiskHigh,
		},
		{
			name:     "edit_file normal",
			toolName: "edit_file",
			params:   map[string]interface{}{"path": "src/main.go"},
			wantRisk: RiskMedium,
		},

		// Delete operations
		{
			name:     "delete_file",
			toolName: "delete_file",
			params:   map[string]interface{}{"path": "temp.txt"},
			wantRisk: RiskHigh,
		},
		{
			name:     "delete_file dangerous",
			toolName: "delete_file",
			params:   map[string]interface{}{"path": "/etc/hosts"},
			wantRisk: RiskHigh,
		},

		// Shell commands
		{
			name:     "run_command safe",
			toolName: "run_command",
			params:   map[string]interface{}{"command": "ls -la"},
			wantRisk: RiskLow,
		},
		{
			name:     "run_command medium",
			toolName: "run_command",
			params:   map[string]interface{}{"command": "npm install"},
			wantRisk: RiskMedium,
		},
		{
			name:     "run_command dangerous",
			toolName: "run_command",
			params:   map[string]interface{}{"command": "rm -rf /tmp"},
			wantRisk: RiskHigh,
		},
		{
			name:     "run_background",
			toolName: "run_background",
			params:   map[string]interface{}{"command": "npm run dev"},
			wantRisk: RiskMedium,
		},

		// Unknown tool
		{
			name:     "unknown tool",
			toolName: "unknown_tool",
			params:   map[string]interface{}{},
			wantRisk: RiskMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyToolRisk(tt.toolName, tt.params)
			if got != tt.wantRisk {
				t.Errorf("ClassifyToolRisk(%q, %v) = %v, want %v", tt.toolName, tt.params, got, tt.wantRisk)
			}
		})
	}
}

func TestIsPathDangerous(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantDanger bool
	}{
		// Dangerous paths
		{
			name:       "/etc/shadow",
			path:       "/etc/shadow",
			wantDanger: true,
		},
		{
			name:       "/etc/passwd",
			path:       "/etc/passwd",
			wantDanger: true,
		},
		{
			name:       "/root directory",
			path:       "/root/file.txt",
			wantDanger: true,
		},
		{
			name:       "/boot directory",
			path:       "/boot/grub/grub.cfg",
			wantDanger: true,
		},
		{
			name:       "root directory",
			path:       "/",
			wantDanger: true,
		},
		{
			name:       "/etc directory",
			path:       "/etc",
			wantDanger: true,
		},

		// Path traversal
		{
			name:       "path traversal ../",
			path:       "../../../etc/passwd",
			wantDanger: true,
		},
		{
			name:       "path traversal .\\",
			path:       "..\\..\\windows\\system32",
			wantDanger: true,
		},

		// Wildcards
		{
			name:       "wildcard *",
			path:       "/tmp/*",
			wantDanger: true,
		},

		// Safe paths
		{
			name:       "home directory file",
			path:       "/home/user/document.txt",
			wantDanger: false,
		},
		{
			name:       "tmp file",
			path:       "/tmp/test.txt",
			wantDanger: false,
		},
		{
			name:       "current directory",
			path:       "./test.txt",
			wantDanger: false,
		},
		{
			name:       "relative path",
			path:       "src/main.go",
			wantDanger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPathDangerous(tt.path)
			if got != tt.wantDanger {
				t.Errorf("IsPathDangerous(%q) = %v, want %v", tt.path, got, tt.wantDanger)
			}
		})
	}
}

func TestEvaluateOperationRisk(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		params   map[string]interface{}
		wantRisk RiskLevel
	}{
		{
			name:     "read single file",
			toolName: "read_file",
			params:   map[string]interface{}{"path": "test.txt"},
			wantRisk: RiskLow,
		},
		{
			name:     "read with wildcard",
			toolName: "read_file",
			params:   map[string]interface{}{"path": "*.txt"},
			wantRisk: RiskMedium, // 批量操作,风险提升
		},
		{
			name:     "list dir recursive",
			toolName: "list_dir",
			params: map[string]interface{}{
				"path":      ".",
				"recursive": true,
			},
			wantRisk: RiskMedium, // 递归操作,风险提升
		},
		{
			name:     "command with pipe",
			toolName: "run_command",
			params:   map[string]interface{}{"command": "cat file.txt | grep pattern"},
			wantRisk: RiskMedium, // 包含管道,风险提升
		},
		{
			name:     "command with redirect",
			toolName: "run_command",
			params:   map[string]interface{}{"command": "echo 'test' > output.txt"},
			wantRisk: RiskMedium, // 包含重定向,风险提升
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateOperationRisk(tt.toolName, tt.params)
			if got != tt.wantRisk {
				t.Errorf("EvaluateOperationRisk(%q, %v) = %v, want %v", tt.toolName, tt.params, got, tt.wantRisk)
			}
		})
	}
}

func TestGetRiskDescription(t *testing.T) {
	tests := []struct {
		name string
		risk RiskLevel
		want string
	}{
		{
			name: "low risk",
			risk: RiskLow,
			want: "低风险操作,通常是只读操作,不会修改系统状态",
		},
		{
			name: "medium risk",
			risk: RiskMedium,
			want: "中等风险操作,可能修改文件或执行命令,需要用户确认",
		},
		{
			name: "high risk",
			risk: RiskHigh,
			want: "高风险操作,可能删除文件、修改系统配置或执行危险命令,强烈建议仔细确认",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRiskDescription(tt.risk)
			if got != tt.want {
				t.Errorf("GetRiskDescription(%v) = %q, want %q", tt.risk, got, tt.want)
			}
		})
	}
}

func TestShouldAutoApprove(t *testing.T) {
	tests := []struct {
		name             string
		risk             RiskLevel
		autoApproveReads bool
		autoApproveAll   bool
		want             bool
	}{
		{
			name:             "auto approve all enabled",
			risk:             RiskHigh,
			autoApproveReads: false,
			autoApproveAll:   true,
			want:             true,
		},
		{
			name:             "auto approve reads - low risk",
			risk:             RiskLow,
			autoApproveReads: true,
			autoApproveAll:   false,
			want:             true,
		},
		{
			name:             "auto approve reads - medium risk",
			risk:             RiskMedium,
			autoApproveReads: true,
			autoApproveAll:   false,
			want:             false,
		},
		{
			name:             "no auto approve",
			risk:             RiskLow,
			autoApproveReads: false,
			autoApproveAll:   false,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldAutoApprove(tt.risk, tt.autoApproveReads, tt.autoApproveAll)
			if got != tt.want {
				t.Errorf("ShouldAutoApprove(%v, %v, %v) = %v, want %v",
					tt.risk, tt.autoApproveReads, tt.autoApproveAll, got, tt.want)
			}
		})
	}
}

func TestRiskLevel_String(t *testing.T) {
	tests := []struct {
		name string
		risk RiskLevel
		want string
	}{
		{
			name: "low",
			risk: RiskLow,
			want: "低",
		},
		{
			name: "medium",
			risk: RiskMedium,
			want: "中",
		},
		{
			name: "high",
			risk: RiskHigh,
			want: "高",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.risk.String()
			if got != tt.want {
				t.Errorf("RiskLevel.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
