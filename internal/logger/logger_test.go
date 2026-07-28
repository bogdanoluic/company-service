package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewWithWriterLogsConfiguredLevel(t *testing.T) {
	var output bytes.Buffer

	log, err := NewWithWriter("debug", &output)
	if err != nil {
		t.Fatalf("NewWithWriter() error = %v", err)
	}

	log.Debug("debug message", "request_id", "23")

	result := output.String()

	if !strings.Contains(result, `"level":"DEBUG"`) {
		t.Errorf("log output does not contain debug level: %s", result)
	}

	if !strings.Contains(result, `"msg":"debug message"`) {
		t.Errorf("log output does not contain message: %s", result)
	}

	if !strings.Contains(result, `"request_id":"23"`) {
		t.Errorf("log output does not contain request ID: %s", result)
	}
}

func TestNewWithWriterFiltersLowerLevels(t *testing.T) {
	var output bytes.Buffer

	log, err := NewWithWriter("info", &output)
	if err != nil {
		t.Fatalf("NewWithWriter() error = %v", err)
	}

	log.Debug("hidden debug message")
	log.Info("visible info message")

	result := output.String()

	if strings.Contains(result, "hidden debug message") {
		t.Errorf("debug message was logged at info level: %s", result)
	}

	if !strings.Contains(result, "visible info message") {
		t.Errorf("info message was not logged: %s", result)
	}
}

func TestNewWithWriterRejectsInvalidLevel(t *testing.T) {
	var output bytes.Buffer

	_, err := NewWithWriter("invalid", &output)
	if err == nil {
		t.Fatal("NewWithWriter() error = nil, want error")
	}
}
