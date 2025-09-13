package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// NewAPIClient creates a new client instance
func NewAPIClient(authConfig *AuthConfig) *APIClient {
	return &APIClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		authConfig: authConfig,
		logger:     NewLogger(authConfig.Debug), // Initialize the logger
	}
}

// doRequest performs an HTTP request
func (c *APIClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) { //////
	token, err := c.getValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	fullURL := c.instanceURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("request failed with status: %s", resp.Status)
		}
		return nil, fmt.Errorf("API error: %s (code: %s)", errResp.Message, errResp.ErrorCode)
	}

	return resp, nil
}

// CreateCase creates a new case with support for custom headers
func (c *APIClient) CreateCase(ctx context.Context, caseData *Case, headers ...CaseHeaders) (*Case, error) {
	// Preparing request headers
	reqHeaders := map[string]string{
		"Content-Type": "application/json",
	}

	// Add the transferred headers if they exist
	if len(headers) > 0 {
		if headers[0].SforceAssignmentRuleHeader != "" {
			reqHeaders["Sforce-Assignment-Rule-Header"] = headers[0].SforceAssignmentRuleHeader
		}
		if headers[0].SforceEmailHeader != "" {
			reqHeaders["Sforce-Email-Header"] = headers[0].SforceEmailHeader
		}
	}

	// Logging request data for debugging
	jsonData, _ := json.MarshalIndent(caseData, "", "  ")
	c.logger.Info("Creating case with data:", string(jsonData))
	c.logger.Info("Request headers:", reqHeaders)

	resp, err := c.doRequestWithHeaders(ctx, "POST", "/services/data/v64.0/sobjects/Case/", caseData, reqHeaders)
	if err != nil {
		// Getting more information about the error
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			c.logger.Error("HTTP error details:", nil, "status", resp.Status, "body", string(body))
			resp.Body.Close()
		}
		return nil, fmt.Errorf("failed to create case: %w", err)
	}
	defer resp.Body.Close()

	// Logging the response
	body, _ := io.ReadAll(resp.Body)
	c.logger.Info("Response received:", "status", resp.Status, "body", string(body))

	// Create a new reader for the response body to decode
	responseReader := bytes.NewReader(body)

	var result Case
	if err := json.NewDecoder(responseReader).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Save the ID of the created case
	if result.ID != "" {
		c.SetCaseID(result.ID)
	}

	return &result, nil
}

// doRequestWithHeaders makes an HTTP request with custom headers
func (c *APIClient) doRequestWithHeaders(ctx context.Context, method, path string, body interface{}, customHeaders map[string]string) (*http.Response, error) {
	token, err := c.getValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	fullURL := c.instanceURL + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Basic Headings
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Adding Custom Headers
	for key, value := range customHeaders {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("request failed with status: %s", resp.Status)
		}
		return nil, fmt.Errorf("API error: %s (code: %s)", errResp.Message, errResp.ErrorCode)
	}

	return resp, nil
}

// CreateAttachment creates an attachment for a case
func (c *APIClient) CreateAttachment(ctx context.Context, filePath string) (map[string]interface{}, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	caseID := c.GetCaseID()
	if caseID == "" {
		return nil, fmt.Errorf("no case ID available, create a case first")
	}

	// Uploading attachment
	res, err := c.UploadAttachment(ctx, caseID, filePath)
	if err != nil {
		// Logging the error
		c.logger.Json(map[string]interface{}{
			"action":  "upload attachment",
			"success": false,
			"case_id": caseID,
			"file":    filepath.Base(filePath),
			"error":   err.Error(),
		})
		return nil, err
	}

	// Logging a successful download
	if success, ok := res["success"].(bool); ok && success {
		if data, exists := res["data"]; exists {
			c.logger.Json(map[string]interface{}{
				"action":  "upload attachment",
				"success": true,
				"case_id": caseID,
				"file":    filepath.Base(filePath),
				"data":    data,
			})
		}
	}

	return res, nil
}

// Query executes a SOQL query
func (c *APIClient) Query(ctx context.Context, soql string) (*QueryResponse, error) {
	path := fmt.Sprintf("/services/data/v64.0/query/?q=%s", url.QueryEscape(soql))

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	return &result, nil
}

// GetCase gets case by ID
func (c *APIClient) GetCase(ctx context.Context, caseID string) (*Case, error) {
	path := fmt.Sprintf("/services/data/v64.0/sobjects/Case/%s", caseID)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Case
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UploadAttachment uploads a file as an attachment to Salesforce
func (c *APIClient) UploadAttachment(ctx context.Context, parentID, filePath string) (map[string]interface{}, error) {
	if parentID == "" {
		return nil, fmt.Errorf("parent ID is required")
	}
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	// Checking the existence of a file
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Getting information about the file
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check file size (Salesforce limit is ~25MB for Attachments)
	if fileInfo.Size() > 25*1024*1024 {
		return nil, fmt.Errorf("file size exceeds 25MB limit: %d bytes", fileInfo.Size())
	}

	// Reading the contents of the file
	rawData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Encode in base64
	b64Data := base64.StdEncoding.EncodeToString(rawData)

	// Getting the file name
	fileName := filepath.Base(filePath)

	// Preparing data for the request
	attachmentData := map[string]interface{}{
		"ParentId": parentID,
		"Name":     fileName,
		"Body":     b64Data,
	}

	// We use the Request method
	res, err := c.Request(
		ctx,
		"/services/data/v58.0/sobjects/Attachment/",
		"POST",
		attachmentData,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	// Checking the response status
	if res.Code >= 400 {
		return nil, fmt.Errorf("attachment upload failed with status: %s", res.Status)
	}

	// Parsing Salesforce response
	var apiResponse struct {
		ID      string `json:"id"`
		Success bool   `json:"success"`
		Errors  []struct {
			Message   string `json:"message"`
			ErrorCode string `json:"errorCode"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(res.Data, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if !apiResponse.Success {
		errorMsg := "Salesforce API error"
		if len(apiResponse.Errors) > 0 {
			errorMsg = fmt.Sprintf("%s: %s (code: %s)", errorMsg, apiResponse.Errors[0].Message, apiResponse.Errors[0].ErrorCode)
		}
		return nil, fmt.Errorf(errorMsg)
	}

	return map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":   apiResponse.ID,
			"name": fileName,
			"size": fileInfo.Size(),
		},
	}, nil
}

// EmailMessage creates a new email message
func (c *APIClient) EmailMessage(ctx context.Context, params EmailMessageParams) (map[string]interface{}, error) {
	// Set default values
	if params.To == "" {
		params.To = "customerservice@acronis.com"
	}
	if params.Status == 0 {
		params.Status = 3
	}

	// If CaseId is not passed, use the value from the client (if any)
	if params.CaseId == "" && c.caseID != "" {
		params.CaseId = c.caseID
	}

	resp, err := c.doRequest(ctx, "POST", "/services/data/v64.0/sobjects/EmailMessage/", params)
	if err != nil {
		return nil, fmt.Errorf("failed to create email message: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
