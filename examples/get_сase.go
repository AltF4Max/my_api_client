package examples

import (
	"context"
	"my-api-client/internal/client"
)

func ExamplesGetCase(ctx context.Context, apiClient *client.APIClient) (*client.Case, error) {
	caseID := "500gK00000JCbtiQAD"
	caseObj, err := apiClient.GetCase(ctx, caseID)
	if err != nil {
		return nil, err //Failed to get case
	}
	return caseObj, nil
}
