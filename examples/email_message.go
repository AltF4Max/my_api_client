package examples

import (
	"context"

	"github.com/AltF4Max/my_api_client/internal/client"
)

func ExamplesEmailMessage(ctx context.Context, apiClient *client.APIClient, to string) (map[string]interface{}, error) {
	// Creating parameters
	params := client.EmailMessageParams{
		ParentId:    "500gK00000JCbtiQAD", // ID case example
		FromAddress: "user@example.com",
		Subject:     "Test Email Subject",
		TextBody:    "This is the body of the test email",
		ToAddress:   to, //config.Salesforce.ToEmail
	}

	// Calling a Method
	result, err := apiClient.EmailMessage(ctx, params)
	if err != nil {
		return nil, err //Error creating email message
	}
	return result, nil
}
