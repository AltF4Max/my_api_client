package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIClient_Request(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RefreshToken: "test-refresh-token-456",
		GrantType:    "refresh_token",
		Debug:        true,
	}

	t.Run("successful POST request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-access-token-123",
					"instance_url": "http://" + r.Host,
					"token_type":   "Bearer",
					"issued_at":    time.Now().Format(time.RFC3339),
				})
				return
			}

			if r.URL.Path == "/services/data/v58.0/sobjects/Attachment/" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":      "00XXXXXXXXXXXXXXX",
					"success": true,
					"errors":  []interface{}{},
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		attachmentData := map[string]interface{}{
			"ParentId": "001XXXXXXXXXXXXXXX",
			"Name":     "test.txt",
			"Body":     "base64-encoded-data",
		}

		ctx := context.Background()
		resp, err := client.Request(
			ctx,
			"/services/data/v58.0/sobjects/Attachment/",
			"POST",
			attachmentData,
			map[string]string{"Content-Type": "application/json"},
		)

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 201, resp.Code)
	})

	t.Run("failed to get valid token - auth error", func(t *testing.T) {
		authErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":             "invalid_grant",
				"error_description": "Invalid refresh token",
			})
		}))
		defer authErrorServer.Close()

		errorConfig := &AuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RefreshToken: "invalid-refresh-token",
			LoginURL:     authErrorServer.URL + "/services/oauth2/token",
			GrantType:    "refresh_token",
			Debug:        true,
		}

		client := NewAPIClient(errorConfig)
		client.SetHTTPClient(authErrorServer.Client())

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/test",
			"GET",
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get valid token")
		assert.Contains(t, err.Error(), "Invalid refresh token")
	})

	t.Run("failed to marshal request data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-token",
					"instance_url": "http://" + r.Host,
					"token_type":   "Bearer",
					"issued_at":    time.Now().Format(time.RFC3339),
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		// Invalid data that cannot be marshaled to JSON
		invalidData := make(chan int)

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/test",
			"POST",
			invalidData,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal request data")
		assert.Contains(t, err.Error(), "unsupported type: chan int")
	})

	t.Run("request failed - network error", func(t *testing.T) {
		// Create client that will fail to connect
		client := NewAPIClient(config)

		// Set HTTP client that will fail all requests
		mockClient := &http.Client{
			Transport: &mockTransport{err: errors.New("network connection failed")},
		}
		client.SetHTTPClient(mockClient)
		client.SetLoginURL("http://invalid-server:9999/services/oauth2/token")
		client.SetInstanceURL("http://invalid-server:9999")

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/test",
			"GET",
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed")
		assert.Contains(t, err.Error(), "network connection failed")
	})

	t.Run("failed to read response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-token",
					"instance_url": "http://" + r.Host,
					"token_type":   "Bearer",
					"issued_at":    time.Now().Format(time.RFC3339),
				})
				return
			}

			if r.URL.Path == "/test" {
				// Returning a response with an unreadable body
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				// We close the connection immediately to cause a read error
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
				}
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/test",
			"GET",
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read response body")
	})

	t.Run("401 response with successful token refresh", func(t *testing.T) {
		authCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				authCount++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": fmt.Sprintf("token-%d", authCount),
					"instance_url": "http://" + r.Host,
					"token_type":   "Bearer",
					"issued_at":    time.Now().Format(time.RFC3339),
				})
				return
			}

			if r.URL.Path == "/test-401" {
				// Return 401 to trigger token refresh
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"message":   "Session expired",
						"errorCode": "INVALID_SESSION_ID",
					},
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		ctx := context.Background()
		resp, err := client.Request(
			ctx,
			"/test-401",
			"GET",
			nil,
			nil,
		)

		// Should return 401 response WITHOUT error (refresh succeeded but response is still 401)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, 401, resp.Code)
		assert.Contains(t, resp.Raw, "Session expired")
	})

	t.Run("401 response with failed token refresh", func(t *testing.T) {
		authCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				authCount++
				if authCount == 1 {
					// First auth succeeds
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"access_token": "first-token",
						"instance_url": "http://" + r.Host,
						"token_type":   "Bearer",
						"issued_at":    time.Now().Format(time.RFC3339),
					})
					return
				} else {
					// Refresh fails
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":             "invalid_grant",
						"error_description": "Refresh token expired",
					})
					return
				}
			}

			if r.URL.Path == "/test-401" {
				// Return 401 to trigger token refresh
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"message":   "Session expired",
						"errorCode": "INVALID_SESSION_ID",
					},
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		ctx := context.Background()
		resp, err := client.Request(
			ctx,
			"/test-401",
			"GET",
			nil,
			nil,
		)

		// Should return 401 response WITHOUT error (even though refresh failed)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, 401, resp.Code)
		assert.Contains(t, resp.Raw, "Session expired")
	})

	t.Run("auth failed with JSON error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":             "invalid_client",
					"error_description": "Invalid client credentials",
				})
				return
			}
		}))
		defer server.Close()

		errorConfig := &AuthConfig{
			ClientID:     "invalid-client-id",
			ClientSecret: "invalid-client-secret",
			RefreshToken: "test-refresh-token",
			LoginURL:     server.URL + "/services/oauth2/token",
			GrantType:    "refresh_token",
			Debug:        true,
		}

		client := NewAPIClient(errorConfig)
		client.SetHTTPClient(server.Client())

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/test",
			"GET",
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get valid token")
		assert.Contains(t, err.Error(), "invalid_client")
		assert.Contains(t, err.Error(), "Invalid client credentials")
	})

	t.Run("auth failed with non-JSON error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
				return
			}
		}))
		defer server.Close()

		errorConfig := &AuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RefreshToken: "test-refresh-token",
			LoginURL:     server.URL + "/services/oauth2/token",
			GrantType:    "refresh_token",
			Debug:        true,
		}

		client := NewAPIClient(errorConfig)
		client.SetHTTPClient(server.Client())

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/test",
			"GET",
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get valid token")
		assert.Contains(t, err.Error(), "auth failed with status: 500 Internal Server Error")
	})

	t.Run("auth failed to decode success response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				// Invalid JSON - missing closing brace
				w.Write([]byte(`{"access_token": "test-token", "instance_url": "http://test.com"`))
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/test",
			"GET",
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get valid token")
		assert.Contains(t, err.Error(), "failed to decode auth response")
	})
	t.Run("API error with invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-token",
					"instance_url": "http://" + r.Host,
					"token_type":   "Bearer",
				})
				return
			}

			if r.URL.Path == "/test-invalid" {
				// Returns invalid JSON
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Invalid", "message": "Something went wrong"`)) // Unclosed parenthesis
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/test-invalid",
			"GET",
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed with status: 400 Bad Request")
	})
	t.Run("API error with single object format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-token",
					"instance_url": "http://" + r.Host,
					"token_type":   "Bearer",
				})
				return
			}

			if r.URL.Path == "/test-forbidden" {
				// Returns a single object instead of an array.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message":   "Access denied",
					"errorCode": "INSUFFICIENT_ACCESS",
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/test-forbidden",
			"GET",
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "API error: Access denied (code: INSUFFICIENT_ACCESS)")
	})
	t.Run("API request failed - invalid URL after auth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/services/oauth2/token" {
				// Authentication was successful, but we are returning an invalid instance_url
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-token",
					"instance_url": "invalid-url://api-server.com", // ← Returning an invalid URL!
					"token_type":   "Bearer",
					"issued_at":    time.Now().Format(time.RFC3339),
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		ctx := context.Background()
		_, err := client.Request(
			ctx,
			"/services/data/v58.0/sobjects/Account/",
			"GET",
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed")
		assert.Contains(t, err.Error(), "unsupported protocol scheme")
	})

}

// Mock transport for network errors
type mockTransport struct {
	err error
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, m.err
}

// Helper methods for testing
func (c *APIClient) SetHTTPClient(client *http.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.httpClient = client
}

func (c *APIClient) SetLoginURL(loginURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.authConfig.LoginURL = loginURL
}

func (c *APIClient) SetInstanceURL(instanceURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instanceURL = instanceURL
}
