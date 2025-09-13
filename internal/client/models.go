package client

import (
	"encoding/json"
	"log"
	"net/http"
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
type Response struct {
	Success bool              `json:"success"`
	Code    int               `json:"code"`
	Status  string            `json:"status"`
	Data    json.RawMessage   `json:"data,omitempty"`
	Raw     string            `json:"raw,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type Logger struct {
	debug bool
}

// NewLogger creates a new logger
func NewLogger(debug bool) *Logger {
	return &Logger{debug: debug}
}

// Info logging information
func (l *Logger) Info(message string, fields ...interface{}) {
	if l.debug {
		if len(fields) > 0 {
			log.Printf("INFO: %s %v", message, fields)
		} else {
			log.Printf("INFO: %s", message)
		}
	}
}

// Warn logging of warnings
func (l *Logger) Warn(message string, fields ...interface{}) {
	if len(fields) > 0 {
		log.Printf("WARN: %s %v", message, fields)
	} else {
		log.Printf("WARN: %s", message)
	}
}

// Error logging errors
func (l *Logger) Error(message string, err error, fields ...interface{}) {
	if err != nil {
		if len(fields) > 0 {
			log.Printf("ERROR: %s - %v %v", message, err, fields)
		} else {
			log.Printf("ERROR: %s - %v", message, err)
		}
	} else {
		if len(fields) > 0 {
			log.Printf("ERROR: %s %v", message, fields)
		} else {
			log.Printf("ERROR: %s", message)
		}
	}
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

func (c *APIClient) SetCaseID(caseID string) {
	if caseID != "" {
		c.caseID = caseID
	}
}

// GetCaseID returns the current case ID
func (c *APIClient) GetCaseID() string {
	return c.caseID
}

// EmailMessageParams contains parameters for creating an EmailMessage
type EmailMessageParams struct {
	CaseId    string `json:"ParentId,omitempty"`
	From      string `json:"FromAddress,omitempty"`
	FromName  string `json:"FromName,omitempty"`
	To        string `json:"ToAddress,omitempty"`
	Subject   string `json:"Subject,omitempty"`
	TextBody  string `json:"TextBody,omitempty"`
	Thread    string `json:"ThreadIdentifier,omitempty"`
	ContactId string `json:"ContactId,omitempty"`
	Status    int    `json:"Status,omitempty"`
}
