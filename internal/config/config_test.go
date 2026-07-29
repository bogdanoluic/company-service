package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

var configurationEnvironmentVariables = []string{
	"SERVER_HOST",
	"SERVER_PORT",
	"SERVER_READ_TIMEOUT",
	"SERVER_WRITE_TIMEOUT",
	"SERVER_IDLE_TIMEOUT",
	"LOG_LEVEL",
}

func TestLoadUsesDefaults(t *testing.T) {
	clearConfigurationEnvironment(t)

	// The password intentionally has no default and must always be provided.
	t.Setenv("DATABASE_PASSWORD", "test-password")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := Config{
		Server: ServerConfig{
			Host:         defaultServerHost,
			Port:         defaultServerPort,
			ReadTimeout:  defaultServerReadTimeout,
			WriteTimeout: defaultServerWriteTimeout,
			IdleTimeout:  defaultServerIdleTimeout,
		},
		Log: LogConfig{
			Level: defaultLogLevel,
		},
	}

	if cfg != want {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	clearConfigurationEnvironment(t)

	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_READ_TIMEOUT", "5s")
	t.Setenv("SERVER_WRITE_TIMEOUT", "15s")
	t.Setenv("SERVER_IDLE_TIMEOUT", "90s")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := Config{
		Server: ServerConfig{
			Host:         "127.0.0.1",
			Port:         9090,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  90 * time.Second,
		},
		Log: LogConfig{
			Level: "debug",
		},
	}

	if cfg != want {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoadRejectsInvalidEnvironmentValues(t *testing.T) {
	tests := []struct {
		name         string
		variableName string
		value        string
		wantError    string
	}{
		{
			name:         "invalid server port",
			variableName: "SERVER_PORT",
			value:        "invalid",
			wantError:    "parse SERVER_PORT",
		},
		{
			name:         "invalid server read timeout",
			variableName: "SERVER_READ_TIMEOUT",
			value:        "invalid",
			wantError:    "parse SERVER_READ_TIMEOUT",
		},
		{
			name:         "invalid server write timeout",
			variableName: "SERVER_WRITE_TIMEOUT",
			value:        "invalid",
			wantError:    "parse SERVER_WRITE_TIMEOUT",
		},
		{
			name:         "invalid server idle timeout",
			variableName: "SERVER_IDLE_TIMEOUT",
			value:        "invalid",
			wantError:    "parse SERVER_IDLE_TIMEOUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigurationEnvironment(t)

			t.Setenv(tt.variableName, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want parsing error")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf(
					"Load() error = %q, want error containing %q",
					err,
					tt.wantError,
				)
			}
		})
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		variableName string
		value        string
		wantError    string
	}{
		{
			name:         "empty server host",
			variableName: "SERVER_HOST",
			value:        "",
			wantError:    "server host is required",
		},
		{
			name:         "server port out of range",
			variableName: "SERVER_PORT",
			value:        "70000",
			wantError:    "server port must be between 1 and 65535",
		},
		{
			name:         "non-positive read timeout",
			variableName: "SERVER_READ_TIMEOUT",
			value:        "0s",
			wantError:    "server read timeout must be positive",
		},
		{
			name:         "invalid log level",
			variableName: "LOG_LEVEL",
			value:        "verbose",
			wantError:    "log level must be debug, info, warn, or error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigurationEnvironment(t)

			t.Setenv(tt.variableName, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want validation error")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf(
					"Load() error = %q, want error containing %q",
					err,
					tt.wantError,
				)
			}
		})
	}
}

func clearConfigurationEnvironment(t *testing.T) {
	t.Helper()

	for _, name := range configurationEnvironmentVariables {
		value, existed := os.LookupEnv(name)

		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}

		t.Cleanup(func() {
			var err error

			if existed {
				err = os.Setenv(name, value)
			} else {
				err = os.Unsetenv(name)
			}

			if err != nil {
				t.Errorf("restore %s: %v", name, err)
			}
		})
	}
}
