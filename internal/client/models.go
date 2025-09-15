package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// AuthResponse model for OAuth response
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	ID          string `json:"id"`
	TokenType   string `json:"token_type"`
	IssuedAt    string `json:"issued_at"`
	Signature   string `json:"signature"`
}

// Case model for Case object with extended fields
type Case struct {
	ID              string `json:"Id,omitempty"`
	Subject         string `json:"Subject,omitempty"`
	Description     string `json:"Description,omitempty"`
	Status          string `json:"Status,omitempty"`
	Priority        string `json:"Priority,omitempty"`
	Origin          string `json:"Origin,omitempty"`
	RecordTypeId    string `json:"RecordTypeId,omitempty"`
	AccountId       string `json:"AccountId,omitempty"`
	ContactId       string `json:"ContactId,omitempty"`
	SuppliedName    string `json:"SuppliedName,omitempty"`
	SuppliedEmail   string `json:"SuppliedEmail,omitempty"`
	SuppliedCountry string `json:"SuppliedCountry__c,omitempty"`
	SuppliedPhone   string `json:"SuppliedPhone,omitempty"`
	IPAddress       string `json:"IP_Address__c,omitempty"`
	Severity        string `json:"Severity__c,omitempty"`
	Product         string `json:"Product__c,omitempty"`
	OperatingSystem string `json:"Operating_System__c,omitempty"`
	WebQueueEmail   string `json:"Web_Queue_Email__c,omitempty"`
	WebURL          string `json:"Web_URL__c,omitempty"`
	Type            string `json:"type,omitempty"`
}

// CaseHeaders headers for creating Case
type CaseHeaders struct {
	SforceAssignmentRuleHeader string `json:"Sforce-Assignment-Rule-Header,omitempty"`
	SforceEmailHeader          string `json:"Sforce-Email-Header,omitempty"`
}

// QueryResponse model for SOQL response
type QueryResponse struct {
	TotalSize int           `json:"totalSize"`
	Done      bool          `json:"done"`
	Records   []interface{} `json:"records"`
}

// ErrorResponse model for API errors
type ErrorResponse struct {
	Message   string   `json:"message"`
	ErrorCode string   `json:"errorCode"`
	Fields    []string `json:"fields,omitempty"`
}

type Response struct {
	Success bool              `json:"success"`
	Code    int               `json:"code"`
	Status  string            `json:"status"`
	Data    json.RawMessage   `json:"data,omitempty"`
	Raw     string            `json:"raw,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// EmailMessageParams contains parameters for creating an EmailMessage
type EmailMessageParams struct {
	ParentId    string `json:"ParentId,omitempty"`
	FromAddress string `json:"FromAddress,omitempty"`
	FromName    string `json:"FromName,omitempty"`
	ToAddress   string `json:"ToAddress,omitempty"`
	Subject     string `json:"Subject,omitempty"`
	TextBody    string `json:"TextBody,omitempty"`
	Status      int    `json:"Status,omitempty"`
}

// AuthConfig authentication configuration
type AuthConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	Username     string
	Password     string
	LoginURL     string
	GrantType    string
	Debug        bool
	ToEmail      string
	LogFile      string
	LogLevel     string
}

// APIClient main client
type APIClient struct {
	httpClient  *http.Client
	authConfig  *AuthConfig
	accessToken string
	instanceURL string
	tokenExpiry time.Time
	caseID      string
	mu          sync.Mutex
	logger      *Logger
}

type Logger struct {
	debug   bool
	logFile *os.File
	writer  io.Writer
}

// NewLogger creates a new logger with file support
func NewLogger(debug bool, logFile string) *Logger {
	var writer io.Writer = os.Stdout

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file %s: %v, using stdout", logFile, err)
		} else {
			writer = file
			return &Logger{debug: debug, logFile: file, writer: writer}
		}
	}

	return &Logger{debug: debug, writer: writer}
}

// Close closes the log file if it's open
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Info logging information
func (l *Logger) Info(message string, fields ...interface{}) {
	if l.debug {
		msg := fmt.Sprintf("INFO: %s", message)
		if len(fields) > 0 {
			msg += fmt.Sprintf(" %v", fields)
		}
		fmt.Fprintln(l.writer, msg)
	}
}

// Warn logging of warnings
func (l *Logger) Warn(message string, fields ...interface{}) {
	msg := fmt.Sprintf("WARN: %s", message)
	if len(fields) > 0 {
		msg += fmt.Sprintf(" %v", fields)
	}
	fmt.Fprintln(l.writer, msg)
}

// Error logging errors
func (l *Logger) Error(message string, err error, fields ...interface{}) {
	msg := fmt.Sprintf("ERROR: %s", message)
	if err != nil {
		msg += fmt.Sprintf(" - %v", err)
	}
	if len(fields) > 0 {
		msg += fmt.Sprintf(" %v", fields)
	}
	fmt.Fprintln(l.writer, msg)
}

// Json logging in JSON format (analog Perl Logger->json)
func (l *Logger) Json(data map[string]interface{}) {
	if l.debug {
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			log.Printf("JSON LOG ERROR: %v", err)
			return
		}
		log.Printf("JSON LOG:\n%s", string(jsonData))
	}
}

func (c *APIClient) Close() error {
	if c.logger != nil {
		return c.logger.Close()
	}
	return nil
}

func (c *APIClient) SetCaseID(caseID string) {
	if caseID != "" {
		c.caseID = caseID
	}
}

// GetCaseID returns the current case ID
func (c *APIClient) GetCaseID() string {
	return c.caseID
}
