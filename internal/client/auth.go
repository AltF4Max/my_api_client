package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// authenticate performs authentication
func (c *APIClient) authenticate(ctx context.Context) error {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", c.authConfig.ClientID)
	data.Set("client_secret", c.authConfig.ClientSecret)
	data.Set("refresh_token", c.authConfig.RefreshToken)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.authConfig.LoginURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	// Improved error handling
	if resp.StatusCode != http.StatusOK {
		// Trying to read the error from response
		var authError struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&authError); err == nil {
			return fmt.Errorf("auth failed: %s - %s", authError.Error, authError.Description)
		}
		return fmt.Errorf("auth failed with status: %s", resp.Status)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.accessToken = authResp.AccessToken
	c.instanceURL = authResp.InstanceURL
	c.tokenExpiry = time.Now().Add(55 * time.Minute)

	if resp.StatusCode == http.StatusOK {
		c.logger.Info("token refreshed successfully",
			"token_expiry", c.tokenExpiry.Format(time.RFC3339),
			"instance_url", c.instanceURL,
		)
	}

	return nil
}

// getValidToken returns a valid token
func (c *APIClient) getValidToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken == "" || time.Until(c.tokenExpiry) < 5*time.Minute {
		if err := c.authenticate(ctx); err != nil {
			return "", err
		}
	}

	return c.accessToken, nil
}

func (c *APIClient) Request(ctx context.Context, path, method string, data interface{}, headers map[string]string) (*Response, error) {
	// We get a valid access token
	token, err := c.getValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid token: %w", err)
	}

	// Forming the full URL
	fullURL := c.instanceURL + path

	// Preparing the request body
	var reqBody []byte
	if data != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		reqBody, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
	}

	// Create an HTTP request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Setting up standard headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(reqBody)))
	}

	// Add custom headers if any
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// We execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Reading the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Convert http.Header to map[string]string
	headerMap := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headerMap[key] = values[0] // Take the first value
		}
	}

	// Create a response object
	response := &Response{
		Success: resp.StatusCode >= 200 && resp.StatusCode < 300,
		Code:    resp.StatusCode,
		Status:  resp.Status,
		Raw:     string(body),
		Headers: headerMap, // We use the converted map
	}

	// Try to decode JSON if possible
	if json.Valid(body) {
		response.Data = body
	}

	// Handling authorization errors
	if resp.StatusCode == 401 {
		c.logger.Warn("authentication failed, attempting token refresh")
		if err := c.forceTokenRefresh(ctx); err != nil {
			return response, fmt.Errorf("token refresh failed: %w", err)
		}
	}

	return response, nil
}

// forceTokenRefresh forces a token refresh
func (c *APIClient) forceTokenRefresh(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.accessToken = ""
	c.tokenExpiry = time.Time{}

	return c.authenticate(ctx)
}

// SetRefreshToken sets a new refresh_token
func (c *APIClient) SetRefreshToken(refreshToken string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.authConfig.RefreshToken = refreshToken
}
