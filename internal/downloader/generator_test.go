package downloader

import (
	"fmt"
	"strings"
	"testing"

	"plexlinker/internal/config"
)

// TestShellEscaping tests the shell escaping for show names with malicious characters
func TestShellEscaping(t *testing.T) {
	tests := []struct {
		name         string
		showName     string
		scriptFormat config.ScriptFormat
		wantContains string // What should be in the output
		wantNotRaw   string // Raw malicious part should not appear unescaped
	}{
		{
			name:         "PowerShell - Single Quote Injection",
			showName:     "Show'; rm -rf /;'",
			scriptFormat: config.ScriptFormatBAT,
			wantContains: "Show''; rm -rf /;''", // Single quotes are doubled inside a PowerShell literal
			wantNotRaw:   "",
		},
		{
			name:         "PowerShell - Command Injection with &",
			showName:     "Show&pause",
			scriptFormat: config.ScriptFormatBAT,
			wantContains: "Show&pause", // & is inert inside a single-quoted PowerShell literal
			wantNotRaw:   "",
		},
		{
			name:         "PowerShell - Double Quote Literal",
			showName:     "Show\"Name",
			scriptFormat: config.ScriptFormatBAT,
			wantContains: "Show\"Name", // Double quotes are inert inside a single-quoted PowerShell literal
			wantNotRaw:   "",
		},
		{
			name:         "Bash - Single Quote Injection",
			showName:     "Show'; rm -rf /;'",
			scriptFormat: config.ScriptFormatSH,
			wantContains: "Show'\\''", // Properly escaped
			wantNotRaw:   "",
		},
		{
			name:         "Bash - Command Substitution",
			showName:     "Show$(whoami)",
			scriptFormat: config.ScriptFormatSH,
			wantContains: "Show$(whoami)", // Should be inside single quotes, so safe
			wantNotRaw:   "",
		},
		{
			name:         "Bash - Backtick Command Substitution",
			showName:     "Show`whoami`",
			scriptFormat: config.ScriptFormatSH,
			wantContains: "Show`whoami`", // Should be inside single quotes, so safe
			wantNotRaw:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: tt.scriptFormat,
			}

			sg := NewScriptGenerator(cfg)
			sg.SetMediaType("shows", tt.showName)

			// Generate the script content
			var script string
			if tt.scriptFormat == config.ScriptFormatBAT {
				script = sg.buildBatchScript("echo test")
			} else {
				script = sg.buildShellScript("echo test")
			}

			// Check that escaped version is present
			if tt.wantContains != "" && !strings.Contains(script, tt.wantContains) {
				t.Errorf("Expected script to contain %q, but it didn't.\nScript:\n%s",
					tt.wantContains, script)
			}

			// Check that raw malicious part is not present (when applicable)
			if tt.wantNotRaw != "" && strings.Contains(script, tt.wantNotRaw) {
				t.Errorf("Script contains unescaped malicious string %q.\nScript:\n%s",
					tt.wantNotRaw, script)
			}
		})
	}
}

// TestURLInjection tests that URLs are properly quoted in aria2c commands
func TestURLInjection(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantQuoted   bool
		wantContains string
	}{
		{
			name:         "Normal URL",
			url:          "https://example.com/file.mkv",
			wantQuoted:   true,
			wantContains: "\"https://example.com/file.mkv\"",
		},
		{
			name:         "URL with spaces",
			url:          "https://example.com/my file.mkv",
			wantQuoted:   true,
			wantContains: "\"https://example.com/my file.mkv\"",
		},
		{
			name:         "URL with semicolon",
			url:          "https://example.com/file;test.mkv",
			wantQuoted:   true,
			wantContains: "\"https://example.com/file;test.mkv\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: config.ScriptFormatBAT,
			}

			sg := NewScriptGenerator(cfg)

			// Check URL is properly formatted with quotes
			cmdString := fmt.Sprintf("aria2c -x4 --http-user %s --http-passwd %s \"%s\";",
				sg.config.Username, sg.config.Password, tt.url)

			if !strings.Contains(cmdString, tt.wantContains) {
				t.Errorf("Expected command to contain %q, but it didn't.\nCommand:\n%s",
					tt.wantContains, cmdString)
			}
		})
	}
}

// TestDestinationPathEscaping tests that destination paths are properly escaped
func TestDestinationPathEscaping(t *testing.T) {
	tests := []struct {
		name         string
		destPath     string
		scriptFormat config.ScriptFormat
		wantContains string
	}{
		{
			name:         "Windows path with spaces",
			destPath:     "W:\\My Movies",
			scriptFormat: config.ScriptFormatBAT,
			wantContains: "'W:\\My Movies'",
		},
		{
			name:         "Unix path with spaces",
			destPath:     "/home/user/My Movies",
			scriptFormat: config.ScriptFormatSH,
			wantContains: "'/home/user/My Movies'",
		},
		{
			name:         "Path with single quote",
			destPath:     "/home/user/John's Movies",
			scriptFormat: config.ScriptFormatSH,
			wantContains: "'/home/user/John'\\''s Movies'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Username:     "testuser",
				Password:     "testpass",
				ScriptFormat: tt.scriptFormat,
			}

			sg := NewScriptGenerator(cfg)

			var script string
			if tt.scriptFormat == config.ScriptFormatBAT {
				script = sg.buildCategoryPowerShellScript("echo test", tt.destPath, "test")
			} else {
				script = sg.buildCategoryShellScript("echo test", tt.destPath, "test")
			}

			if !strings.Contains(script, tt.wantContains) {
				t.Errorf("Expected script to contain %q, but it didn't.\nScript:\n%s",
					tt.wantContains, script)
			}
		})
	}
}
