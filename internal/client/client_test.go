package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIClient_CreateCase(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh-token",
		GrantType:    "refresh_token",
		Debug:        true,
	}

	t.Run("successful case creation", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/sobjects/Case/" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":          "500XXXXXXXXXXXXXXX",
					"success":     true,
					"caseNumber":  "00001000",
					"subject":     "Test Case",
					"description": "Test description",
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		caseData := &Case{
			Subject:     "Test Case",
			Description: "Test description",
			Priority:    "High",
		}

		ctx := context.Background()
		result, err := client.CreateCase(ctx, caseData)

		require.NoError(t, err)
		assert.Equal(t, "500XXXXXXXXXXXXXXX", result.ID)
		assert.Equal(t, "00001000", result.CaseNumber)
		assert.Equal(t, "Test Case", result.Subject)
		assert.Equal(t, "test-token", client.accessToken)
	})

	t.Run("case creation with custom headers", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/sobjects/Case/" {
				// Check custom headers
				assert.Equal(t, "assignment-rule-123", r.Header.Get("Sforce-Assignment-Rule-Header"))
				assert.Equal(t, "true", r.Header.Get("Sforce-Email-Header"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":      "500XXXXXXXXXXXXXXX",
					"success": true,
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		caseData := &Case{Subject: "Test Case"}
		headers := CaseHeaders{
			SforceAssignmentRuleHeader: "assignment-rule-123",
			SforceEmailHeader:          "true",
		}

		ctx := context.Background()
		_, err := client.CreateCase(ctx, caseData, headers)

		require.NoError(t, err)
	})

	t.Run("case creation with API error", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/sobjects/Case/" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"message":   "Required fields are missing",
						"errorCode": "REQUIRED_FIELD_MISSING",
						"fields":    []string{"Subject"},
					},
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		caseData := &Case{} // Missing required Subject field

		ctx := context.Background()
		_, err := client.CreateCase(ctx, caseData)

		require.Error(t, err)
		// Updating the expected error message
		assert.Contains(t, err.Error(), "API error: Required fields are missing (code: REQUIRED_FIELD_MISSING)")
	})
}

func TestAPIClient_Query(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh-token",
		GrantType:    "refresh_token",
		Debug:        true,
	}

	t.Run("successful query", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/query/" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"totalSize": 1,
					"done":      true,
					"records": []map[string]interface{}{
						{
							"Id":         "500XXXXXXXXXXXXXXX",
							"CaseNumber": "00001000",
							"Subject":    "Test Case",
							"Status":     "New",
							"attributes": map[string]interface{}{
								"type": "Case",
							},
						},
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
		soql := "SELECT Id, CaseNumber, Subject FROM Case WHERE Status = 'New'"
		result, err := client.Query(ctx, soql)

		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalSize)
		assert.True(t, result.Done)
		assert.Len(t, result.Records, 1)
	})

	t.Run("query with API error", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/query/" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"message":   "Malformed query",
						"errorCode": "MALFORMED_QUERY",
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
		soql := "INVALID QUERY SYNTAX"
		_, err := client.Query(ctx, soql)

		require.Error(t, err)
		// Updating the expected error message
		assert.Contains(t, err.Error(), "API error: Malformed query (code: MALFORMED_QUERY)")
	})
}

func TestAPIClient_GetCase(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh-token",
		GrantType:    "refresh_token",
		Debug:        true,
	}

	t.Run("successful get case", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/sobjects/Case/500XXXXXXXXXXXXXXX" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"Id":         "500XXXXXXXXXXXXXXX",
					"CaseNumber": "00001000",
					"Subject":    "Test Case",
					"Status":     "New",
					"Priority":   "High",
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		ctx := context.Background()
		caseID := "500XXXXXXXXXXXXXXX"
		result, err := client.GetCase(ctx, caseID)

		require.NoError(t, err)
		assert.Equal(t, "500XXXXXXXXXXXXXXX", result.ID)
		assert.Equal(t, "00001000", result.CaseNumber)
		assert.Equal(t, "Test Case", result.Subject)
		assert.Equal(t, "New", result.Status)
	})

	t.Run("get case not found", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/sobjects/Case/INVALID_CASE_ID" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"message":   "The requested resource does not exist",
						"errorCode": "NOT_FOUND",
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
		caseID := "INVALID_CASE_ID"
		_, err := client.GetCase(ctx, caseID)

		require.Error(t, err)
		// Updating the expected error message
		assert.Contains(t, err.Error(), "API error: The requested resource does not exist (code: NOT_FOUND)")
	})
}

func TestAPIClient_EmailMessage(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh-token",
		GrantType:    "refresh_token",
		Debug:        true,
	}

	t.Run("successful email message creation", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/sobjects/EmailMessage/" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":      "00XXXXXXXXXXXXXXX",
					"success": true,
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		params := EmailMessageParams{
			ParentId:    "500XXXXXXXXXXXXXXX",
			FromAddress: "test@example.com",
			ToAddress:   "recipient@example.com",
			Subject:     "Test Subject",
			TextBody:    "Test message body",
			Status:      3,
		}

		ctx := context.Background()
		result, err := client.EmailMessage(ctx, params)

		require.NoError(t, err)
		assert.Equal(t, "00XXXXXXXXXXXXXXX", result["id"])
		assert.Equal(t, true, result["success"])
	})

	t.Run("email message uses case ID from client", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/sobjects/EmailMessage/" {
				// Read and check the request body
				var requestData EmailMessageParams
				json.NewDecoder(r.Body).Decode(&requestData)
				assert.Equal(t, "500XXXXXXXXXXXXXXX", requestData.ParentId)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":      "00XXXXXXXXXXXXXXX",
					"success": true,
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")
		client.SetCaseID("500XXXXXXXXXXXXXXX") // Set case ID on client

		params := EmailMessageParams{
			FromAddress: "test@example.com",
			ToAddress:   "recipient@example.com",
			Subject:     "Test Subject",
			TextBody:    "Test message body",
			// ParentId is empty, should use client's case ID
		}

		ctx := context.Background()
		_, err := client.EmailMessage(ctx, params)

		require.NoError(t, err)
	})

	t.Run("email message creation with API error", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v64.0/sobjects/EmailMessage/" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"message":   "Invalid email address",
						"errorCode": "INVALID_EMAIL_ADDRESS",
					},
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		params := EmailMessageParams{
			ParentId:    "500XXXXXXXXXXXXXXX",
			FromAddress: "invalid-email",
			ToAddress:   "recipient@example.com",
			Subject:     "Test Subject",
			TextBody:    "Test message body",
		}

		ctx := context.Background()
		_, err := client.EmailMessage(ctx, params)

		require.Error(t, err)
		// Updating the expected error message
		assert.Contains(t, err.Error(), "API error: Invalid email address (code: INVALID_EMAIL_ADDRESS)")
	})
}

func TestAPIClient_UploadAttachment(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh-token",
		GrantType:    "refresh_token",
		Debug:        true,
	}

	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test-file-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("test file content")
	require.NoError(t, err)
	tmpFile.Close()

	t.Run("upload attachment validation errors", func(t *testing.T) {
		client := NewAPIClient(config)

		ctx := context.Background()

		// Test empty parent ID
		_, err := client.UploadAttachment(ctx, "", "test.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parent ID is required")

		// Test empty file path
		_, err = client.UploadAttachment(ctx, "500XXXXXXXXXXXXXXX", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file path is required")
	})

	t.Run("file not found error", func(t *testing.T) {
		client := NewAPIClient(config)

		ctx := context.Background()
		_, err := client.UploadAttachment(ctx, "500XXXXXXXXXXXXXXX", "nonexistent.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "file does not exist")
	})

	t.Run("successful attachment upload", func(t *testing.T) {
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

		ctx := context.Background()
		result, err := client.UploadAttachment(ctx, "500XXXXXXXXXXXXXXX", tmpFile.Name())

		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result, "data")
	})

	t.Run("attachment upload with API error", func(t *testing.T) {
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

			if r.URL.Path == "/services/data/v58.0/sobjects/Attachment/" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				// Return in array format (standard for Salesforce)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"message":   "Parent ID does not exist",
						"errorCode": "ENTITY_IS_DELETED",
						"fields":    []string{"ParentId"},
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
		_, err := client.UploadAttachment(ctx, "INVALID_PARENT_ID", tmpFile.Name())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "API error: Parent ID does not exist (code: ENTITY_IS_DELETED)")
	})
}

func TestAPIClient_CreateAttachment(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh-token",
		GrantType:    "refresh_token",
		Debug:        true,
	}

	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test-file-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("test file content")
	require.NoError(t, err)
	tmpFile.Close()

	t.Run("create attachment without case ID", func(t *testing.T) {
		client := NewAPIClient(config)

		ctx := context.Background()
		_, err := client.CreateAttachment(ctx, tmpFile.Name())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no case ID available")
	})

	t.Run("successful attachment creation", func(t *testing.T) {
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
		client.SetCaseID("500XXXXXXXXXXXXXXX") // Set case ID

		ctx := context.Background()
		result, err := client.CreateAttachment(ctx, tmpFile.Name())

		require.NoError(t, err)
		assert.True(t, result["success"].(bool))
		assert.Contains(t, result, "data")
	})
}

func TestAPIClient_doRequest(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh-token",
		GrantType:    "refresh_token",
		Debug:        true,
	}

	t.Run("doRequest with API error response", func(t *testing.T) {
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

			if r.URL.Path == "/test-error" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"message":   "Invalid request",
						"errorCode": "INVALID_REQUEST",
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
		_, err := client.doRequest(ctx, "GET", "/test-error", nil)

		require.Error(t, err)
		// Updating the expected error message
		assert.Contains(t, err.Error(), "API error: Invalid request (code: INVALID_REQUEST)")
	})
}

func TestAPIClient_doRequestWithHeaders(t *testing.T) {
	config := &AuthConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh-token",
		GrantType:    "refresh_token",
		Debug:        true,
	}

	t.Run("doRequestWithHeaders with custom headers", func(t *testing.T) {
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

			if r.URL.Path == "/test-headers" {
				// Check custom headers
				assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
				assert.Equal(t, "another-value", r.Header.Get("X-Another-Header"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
				})
				return
			}
		}))
		defer server.Close()

		client := NewAPIClient(config)
		client.SetHTTPClient(server.Client())
		client.SetLoginURL(server.URL + "/services/oauth2/token")

		ctx := context.Background()
		headers := map[string]string{
			"X-Custom-Header":  "custom-value",
			"X-Another-Header": "another-value",
		}

		resp, err := client.doRequestWithHeaders(ctx, "GET", "/test-headers", nil, headers)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
