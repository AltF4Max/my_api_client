package client

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLogger_NewLogger tests Logger constructor
func TestLogger_NewLogger(t *testing.T) {
	tests := []struct {
		name     string
		debug    bool
		expected bool
	}{
		{"Debug enabled", true, true},
		{"Debug disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.debug)
			if logger.debug != tt.expected {
				t.Errorf("Expected debug=%v, got=%v", tt.expected, logger.debug)
			}
		})
	}
}

// TestLogger_Info tests Info logging
func TestLogger_Info(t *testing.T) {
	originalOutput := log.Writer()
	defer log.SetOutput(originalOutput)

	var buf bytes.Buffer
	log.SetOutput(&buf)

	tests := []struct {
		name     string
		debug    bool
		message  string
		fields   []interface{}
		expected string
	}{
		{"Debug enabled with fields", true, "test message", []interface{}{"field1", 123}, "INFO: test message [field1 123]"},
		{"Debug enabled no fields", true, "simple message", nil, "INFO: simple message"},
		{"Debug disabled", false, "should not appear", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger := &Logger{debug: tt.debug}
			logger.Info(tt.message, tt.fields...)

			output := buf.String()
			if tt.debug && tt.expected != "" && !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got: '%s'", tt.expected, output)
			}
			if !tt.debug && output != "" {
				t.Errorf("Expected no output when debug disabled, got: '%s'", output)
			}
		})
	}
}

// TestLogger_Warn tests Warn logging
func TestLogger_Warn(t *testing.T) {
	originalOutput := log.Writer()
	defer log.SetOutput(originalOutput)

	var buf bytes.Buffer
	log.SetOutput(&buf)

	tests := []struct {
		name     string
		message  string
		fields   []interface{}
		expected string
	}{
		{"With fields", "warning", []interface{}{"data", 456}, "WARN: warning [data 456]"},
		{"No fields", "simple warning", nil, "WARN: simple warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger := &Logger{debug: true} // Warn always logs, regardless of debug
			logger.Warn(tt.message, tt.fields...)

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got: '%s'", tt.expected, output)
			}
		})
	}
}

// TestLogger_Error tests Error logging
func TestLogger_Error(t *testing.T) {
	originalOutput := log.Writer()
	defer log.SetOutput(originalOutput)

	var buf bytes.Buffer
	log.SetOutput(&buf)

	testError := errors.New("test error")

	tests := []struct {
		name     string
		message  string
		err      error
		fields   []interface{}
		expected string
	}{
		{"With error and fields", "operation failed", testError, []interface{}{"context", "value"}, "ERROR: operation failed - test error [context value]"},
		{"With error no fields", "failed", testError, nil, "ERROR: failed - test error"},
		{"No error with fields", "issue", nil, []interface{}{"detail", 789}, "ERROR: issue [detail 789]"},
		{"No error no fields", "problem", nil, nil, "ERROR: problem"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger := &Logger{debug: true} // Error is always logged
			logger.Error(tt.message, tt.err, tt.fields...)

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got: '%s'", tt.expected, output)
			}
		})
	}
}

// TestLogger_Json tests JSON logging
func TestLogger_Json(t *testing.T) {
	originalOutput := log.Writer()
	defer log.SetOutput(originalOutput)

	var buf bytes.Buffer
	log.SetOutput(&buf)

	tests := []struct {
		name      string
		debug     bool
		data      map[string]interface{}
		contains  string
		shouldLog bool
	}{
		{"Debug enabled with valid data", true, map[string]interface{}{"key": "value"}, "JSON LOG:", true},
		{"Debug disabled", false, map[string]interface{}{"key": "value"}, "", false},
		{"Empty data", true, map[string]interface{}{}, "{}", true},
		{"Nested data", true, map[string]interface{}{"user": map[string]interface{}{"name": "test"}}, "user", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger := &Logger{debug: tt.debug}
			logger.Json(tt.data)

			output := buf.String()
			if tt.shouldLog && !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: '%s'", tt.contains, output)
			}
			if !tt.shouldLog && output != "" {
				t.Errorf("Expected no output, got: '%s'", output)
			}
		})
	}
}

// TestLogger_Json_Error tests JSON logging error handling
func TestLogger_Json_Error(t *testing.T) {
	originalOutput := log.Writer()
	defer log.SetOutput(originalOutput)

	var buf bytes.Buffer
	log.SetOutput(&buf)

	logger := &Logger{debug: true}
	// Create invalid JSON data (channel cannot be marshaled)
	invalidData := map[string]interface{}{
		"channel": make(chan int),
	}

	logger.Json(invalidData)

	output := buf.String()
	if !strings.Contains(output, "JSON LOG ERROR:") {
		t.Errorf("Expected error message, got: '%s'", output)
	}
}

// TestAPIClient_SetCaseID tests SetCaseID method
func TestAPIClient_SetCaseID(t *testing.T) {
	client := &APIClient{}

	// Test setting non-empty case ID
	client.SetCaseID("test-case-123")
	if client.GetCaseID() != "test-case-123" {
		t.Errorf("Expected case ID 'test-case-123', got '%s'", client.GetCaseID())
	}

	// Test setting empty case ID (should not change)
	client.SetCaseID("")
	if client.GetCaseID() != "test-case-123" {
		t.Errorf("Expected case ID to remain 'test-case-123', got '%s'", client.GetCaseID())
	}

	// Test setting new case ID
	client.SetCaseID("new-case-456")
	if client.GetCaseID() != "new-case-456" {
		t.Errorf("Expected case ID 'new-case-456', got '%s'", client.GetCaseID())
	}
}

// TestAPIClient_GetCaseID tests GetCaseID method
func TestAPIClient_GetCaseID(t *testing.T) {
	client := &APIClient{}

	// Test default value (should be empty)
	if client.GetCaseID() != "" {
		t.Errorf("Expected empty case ID, got '%s'", client.GetCaseID())
	}

	// Test after setting value
	client.caseID = "test-case"
	if client.GetCaseID() != "test-case" {
		t.Errorf("Expected case ID 'test-case', got '%s'", client.GetCaseID())
	}
}

// TestAuthConfig_Structure tests AuthConfig structure
func TestAuthConfig_Structure(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client",
		ClientSecret: "secret",
		RefreshToken: "refresh-token",
		Username:     "user",
		Password:     "pass",
		LoginURL:     "https://login.example.com",
		GrantType:    "password",
		Debug:        true,
	}

	if config.ClientID != "test-client" {
		t.Errorf("Expected ClientID 'test-client', got '%s'", config.ClientID)
	}
	if config.ClientSecret != "secret" {
		t.Errorf("Expected ClientSecret 'secret', got '%s'", config.ClientSecret)
	}
	if !config.Debug {
		t.Error("Expected Debug to be true")
	}
}

// TestAPIClient_Structure tests APIClient structure
func TestAPIClient_Structure(t *testing.T) {
	authConfig := &AuthConfig{ClientID: "test-client"}
	logger := NewLogger(true)

	client := &APIClient{
		httpClient:  &http.Client{},
		authConfig:  authConfig,
		accessToken: "token-123",
		instanceURL: "https://instance.example.com",
		tokenExpiry: time.Now().Add(1 * time.Hour),
		caseID:      "test-case",
		mu:          sync.Mutex{},
		logger:      logger,
	}

	if client.authConfig.ClientID != "test-client" {
		t.Errorf("Expected authConfig ClientID 'test-client', got '%s'", client.authConfig.ClientID)
	}
	if client.accessToken != "token-123" {
		t.Errorf("Expected accessToken 'token-123', got '%s'", client.accessToken)
	}
	if client.caseID != "test-case" {
		t.Errorf("Expected caseID 'test-case', got '%s'", client.caseID)
	}
	if client.logger == nil {
		t.Error("Expected logger to be set")
	}
}
