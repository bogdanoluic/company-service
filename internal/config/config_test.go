package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	configPath := writeTestConfig(t, `{
		"server": {
			"host": "127.0.0.1",
			"port": 8080,
			"read_timeout": "5s",
			"write_timeout": "10s",
			"idle_timeout": "60s"
		},
		"log": {
			"level": "info"
		}
	}`)

	t.Setenv("CONFIG_PATH", configPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Server.Port, 8080)
	}

	if cfg.Log.Level != "info" {
		t.Errorf("Log level = %q, want %q", cfg.Log.Level, "info")
	}
}

func TestLoadEnvironmentOverrides(t *testing.T) {
	configPath := writeTestConfig(t, `{
		"server": {
			"host": "127.0.0.1",
			"port": 8080,
			"read_timeout": "5s",
			"write_timeout": "10s",
			"idle_timeout": "60s"
		},
		"log": {
			"level": "info"
		}
	}`)

	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("SERVER_HOST", "0.0.0.0")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Port = %d, want %d", cfg.Server.Port, 9090)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("Log level = %q, want %q", cfg.Log.Level, "debug")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	configPath := writeTestConfig(t, `{
		"server": {
			"host": "127.0.0.1",
			"port": 70000,
			"read_timeout": "5s",
			"write_timeout": "10s",
			"idle_timeout": "60s"
		},
		"log": {
			"level": "info"
		}
	}`)

	t.Setenv("CONFIG_PATH", configPath)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestLoadRejectsInvalidEnvironmentPort(t *testing.T) {
	configPath := writeTestConfig(t, `{
		"server": {
			"host": "127.0.0.1",
			"port": 8080,
			"read_timeout": "5s",
			"write_timeout": "10s",
			"idle_timeout": "60s"
		},
		"log": {
			"level": "info"
		}
	}`)

	t.Setenv("CONFIG_PATH", configPath)
	t.Setenv("SERVER_PORT", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want parsing error")
	}
}

func writeTestConfig(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write test configuration: %v", err)
	}

	return path
}
