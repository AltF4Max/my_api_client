package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLogger_NewLogger tests Logger constructor
func TestLogger_NewLogger(t *testing.T) {
	tests := []struct {
		name      string
		debug     bool
		logFile   string
		expectErr bool
	}{
		{"Debug enabled, no file", true, "", false},
		{"Debug disabled, no file", false, "", false},
		{"Debug enabled, with file", true, "test.log", false},
		{"Debug disabled, with file", false, "test.log", false},
		{"Debug enabled, invalid file", true, "/invalid/path/test.log", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if tt.logFile != "" {
					os.Remove(tt.logFile)
				}
			}()

			logger := NewLogger(tt.debug, tt.logFile)
			if logger == nil {
				t.Fatal("Expected logger to be created, got nil")
			}

			if logger.debug != tt.debug {
				t.Errorf("Expected debug=%v, got=%v", tt.debug, logger.debug)
			}

			if tt.logFile != "" && !tt.expectErr && logger.logFile == nil {
				t.Error("Expected log file to be opened")
			}

			// Clean up
			if logger.logFile != nil {
				logger.Close()
			}
		})
	}
}

// TestLogger_Close tests Close method
func TestLogger_Close(t *testing.T) {
	logger := NewLogger(true, "test_close.log")
	defer os.Remove("test_close.log")

	if err := logger.Close(); err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}

	// Test double close - should return nil or a specific error
	if err := logger.Close(); err != nil {
		// We check that this is the expected "file already closed" error
		if !strings.Contains(err.Error(), "already closed") {
			t.Errorf("Expected 'already closed' error, got: %v", err)
		}
	}
}

// TestLogger_Info tests Info logging
func TestLogger_Info(t *testing.T) {
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
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := NewLogger(tt.debug, "")
			logger.Info(tt.message, tt.fields...)
			logger.Close()

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
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
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := NewLogger(true, "") // Warn always logs, regardless of debug
			logger.Warn(tt.message, tt.fields...)
			logger.Close()

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got: '%s'", tt.expected, output)
			}
		})
	}
}

// TestLogger_Error tests Error logging
func TestLogger_Error(t *testing.T) {
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
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := NewLogger(true, "") // Error is always logged
			logger.Error(tt.message, tt.err, tt.fields...)
			logger.Close()

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got: '%s'", tt.expected, output)
			}
		})
	}
}

// TestLogger_Json tests JSON logging
func TestLogger_Json(t *testing.T) {
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
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := NewLogger(tt.debug, "")
			logger.Json(tt.data)
			logger.Close()

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			io.Copy(&buf, r)
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
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := NewLogger(true, "")
	// Create invalid JSON data (channel cannot be marshaled)
	invalidData := map[string]interface{}{
		"channel": make(chan int),
	}

	logger.Json(invalidData)
	logger.Close()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// We update the expected string - it is now the error message from l.Error()
	if !strings.Contains(output, "ERROR: JSON marshaling failed") {
		t.Errorf("Expected error message 'ERROR: JSON marshaling failed', got: '%s'", output)
	}
}

// TestAPIClient_SetCaseID tests SetCaseID method
func TestAPIClient_SetCaseID(t *testing.T) {
	logger := NewLogger(false, "")
	defer logger.Close()

	client := &APIClient{
		logger: logger,
	}

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
	logger := NewLogger(false, "")
	defer logger.Close()

	client := &APIClient{
		logger: logger,
	}

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
		ToEmail:      "test@example.com",
		LogFile:      "app.log",
		LogLevel:     "info",
	}

	if config.ClientID != "test-client" {
		t.Errorf("Expected ClientID 'test-client', got '%s'", config.ClientID)
	}
	if config.ClientSecret != "secret" {
		t.Errorf("Expected ClientSecret 'secret', got '%s'", config.ClientSecret)
	}
	if config.ToEmail != "test@example.com" {
		t.Errorf("Expected ToEmail 'test@example.com', got '%s'", config.ToEmail)
	}
	if !config.Debug {
		t.Error("Expected Debug to be true")
	}
}

// TestAPIClient_Structure tests APIClient structure
func TestAPIClient_Structure(t *testing.T) {
	authConfig := &AuthConfig{ClientID: "test-client"}
	logger := NewLogger(true, "")
	defer logger.Close()

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

// TestAPIClient_Close tests Close method
func TestAPIClient_Close(t *testing.T) {
	logger := NewLogger(false, "")
	client := &APIClient{
		logger: logger,
	}

	if err := client.Close(); err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}

	// Test with nil logger
	client.logger = nil
	if err := client.Close(); err != nil {
		t.Errorf("Expected no error with nil logger, got: %v", err)
	}
}

// TestResponse_Structure tests Response structure
func TestResponse_Structure(t *testing.T) {
	response := &Response{
		Success: true,
		Code:    200,
		Status:  "OK",
		Data:    json.RawMessage(`{"key": "value"}`),
		Raw:     "raw response",
		Headers: map[string]string{"Content-Type": "application/json"},
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}
	if response.Code != 200 {
		t.Errorf("Expected Code 200, got %d", response.Code)
	}
	if response.Status != "OK" {
		t.Errorf("Expected Status 'OK', got '%s'", response.Status)
	}
}

// TestEmailMessageParams_Structure tests EmailMessageParams structure
func TestEmailMessageParams_Structure(t *testing.T) {
	params := &EmailMessageParams{
		ParentId:    "001xx000003DG",
		FromAddress: "from@example.com",
		FromName:    "Sender",
		ToAddress:   "to@example.com",
		Subject:     "Test Subject",
		TextBody:    "Test body",
		Status:      1,
		Incoming:    true,
	}

	if params.ParentId != "001xx000003DG" {
		t.Errorf("Expected ParentId '001xx000003DG', got '%s'", params.ParentId)
	}
	if params.FromAddress != "from@example.com" {
		t.Errorf("Expected FromAddress 'from@example.com', got '%s'", params.FromAddress)
	}
	if !params.Incoming {
		t.Error("Expected Incoming to be true")
	}
}
