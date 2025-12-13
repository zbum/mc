package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "expand tilde",
			input:    "~/.ssh/id_rsa",
			expected: filepath.Join(home, ".ssh/id_rsa"),
		},
		{
			name:     "absolute path unchanged",
			input:    "/etc/ssh/config",
			expected: "/etc/ssh/config",
		},
		{
			name:     "relative path unchanged",
			input:    "config/test",
			expected: "config/test",
		},
		{
			name:     "tilde only in middle unchanged",
			input:    "/path/~/file",
			expected: "/path/~/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSSHHostDisplay(t *testing.T) {
	tests := []struct {
		name     string
		host     SSHHost
		contains []string
	}{
		{
			name: "full host info",
			host: SSHHost{
				Name:     "production",
				HostName: "prod.example.com",
				User:     "admin",
				Port:     "2222",
				Comment:  "Production server",
			},
			contains: []string{"production", "prod.example.com", "admin", "2222", "Production server"},
		},
		{
			name: "minimal host info",
			host: SSHHost{
				Name: "test-server",
				Port: "22",
			},
			contains: []string{"test-server"},
		},
		{
			name: "default port not shown",
			host: SSHHost{
				Name:     "server",
				HostName: "server.example.com",
				Port:     "22",
			},
			contains: []string{"server", "server.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.host.Display()
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("Display() = %q, should contain %q", result, substr)
				}
			}
			// default port should not be shown
			if tt.host.Port == "22" && strings.Contains(result, "port=22") {
				t.Errorf("Display() = %q, should not contain port=22 for default port", result)
			}
		})
	}
}

func TestParseSSHConfig(t *testing.T) {
	// 임시 SSH config 파일 생성
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	configContent := `# Production server
Host prod
    HostName prod.example.com
    User admin
    Port 2222
    IdentityFile ~/.ssh/prod_key

# Development server
Host dev
    HostName dev.example.com
    User developer

Host staging
    HostName staging.example.com
    Port 22

# Wildcard should be ignored
Host *
    ServerAliveInterval 60
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	hosts, err := parseSSHConfig(configPath)
	if err != nil {
		t.Fatalf("parseSSHConfig() error = %v", err)
	}

	if len(hosts) != 3 {
		t.Errorf("parseSSHConfig() returned %d hosts, want 3", len(hosts))
	}

	// prod 호스트 검증
	prod := findHost(hosts, "prod")
	if prod == nil {
		t.Fatal("prod host not found")
	}
	if prod.HostName != "prod.example.com" {
		t.Errorf("prod.HostName = %q, want %q", prod.HostName, "prod.example.com")
	}
	if prod.User != "admin" {
		t.Errorf("prod.User = %q, want %q", prod.User, "admin")
	}
	if prod.Port != "2222" {
		t.Errorf("prod.Port = %q, want %q", prod.Port, "2222")
	}
	if prod.Comment != "Production server" {
		t.Errorf("prod.Comment = %q, want %q", prod.Comment, "Production server")
	}

	// dev 호스트 검증
	dev := findHost(hosts, "dev")
	if dev == nil {
		t.Fatal("dev host not found")
	}
	if dev.HostName != "dev.example.com" {
		t.Errorf("dev.HostName = %q, want %q", dev.HostName, "dev.example.com")
	}
	if dev.Port != "22" {
		t.Errorf("dev.Port = %q, want %q (default)", dev.Port, "22")
	}

	// staging 호스트 검증
	staging := findHost(hosts, "staging")
	if staging == nil {
		t.Fatal("staging host not found")
	}

	// 와일드카드 호스트가 포함되지 않았는지 확인
	wildcard := findHost(hosts, "*")
	if wildcard != nil {
		t.Error("wildcard host should not be included")
	}
}

func TestParseSSHConfigWithEquals(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// = 구분자를 사용하는 config
	configContent := `Host myserver
    HostName=myserver.example.com
    User=myuser
    Port=3022
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	hosts, err := parseSSHConfig(configPath)
	if err != nil {
		t.Fatalf("parseSSHConfig() error = %v", err)
	}

	if len(hosts) != 1 {
		t.Fatalf("parseSSHConfig() returned %d hosts, want 1", len(hosts))
	}

	host := hosts[0]
	if host.HostName != "myserver.example.com" {
		t.Errorf("HostName = %q, want %q", host.HostName, "myserver.example.com")
	}
	if host.User != "myuser" {
		t.Errorf("User = %q, want %q", host.User, "myuser")
	}
	if host.Port != "3022" {
		t.Errorf("Port = %q, want %q", host.Port, "3022")
	}
}

func TestParseSSHConfigEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	hosts, err := parseSSHConfig(configPath)
	if err != nil {
		t.Fatalf("parseSSHConfig() error = %v", err)
	}

	if len(hosts) != 0 {
		t.Errorf("parseSSHConfig() returned %d hosts, want 0", len(hosts))
	}
}

func TestParseSSHConfigNotFound(t *testing.T) {
	_, err := parseSSHConfig("/nonexistent/path/config")
	if err == nil {
		t.Error("parseSSHConfig() should return error for nonexistent file")
	}
}

func TestGetSSHConfigPath(t *testing.T) {
	path := getSSHConfigPath()
	if path == "" {
		t.Skip("cannot get home directory")
	}

	if !strings.HasSuffix(path, ".ssh/config") {
		t.Errorf("getSSHConfigPath() = %q, should end with .ssh/config", path)
	}
}

func TestGetDefaultKeyPaths(t *testing.T) {
	paths := getDefaultKeyPaths()
	if paths == nil {
		t.Skip("cannot get home directory")
	}

	expectedKeys := []string{"id_ed25519", "id_rsa", "id_ecdsa", "id_dsa"}
	if len(paths) != len(expectedKeys) {
		t.Errorf("getDefaultKeyPaths() returned %d paths, want %d", len(paths), len(expectedKeys))
	}

	for i, expected := range expectedKeys {
		if !strings.HasSuffix(paths[i], expected) {
			t.Errorf("paths[%d] = %q, should end with %q", i, paths[i], expected)
		}
	}
}

// Helper function
func findHost(hosts []SSHHost, name string) *SSHHost {
	for _, h := range hosts {
		if h.Name == name {
			return &h
		}
	}
	return nil
}