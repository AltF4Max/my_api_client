package examples

import (
	"context"

	"github.com/AltF4Max/sf_api_client/internal/client"
)

func ExamplesGetCase(ctx context.Context, apiClient *client.APIClient) (*client.Case, error) {
	caseID := "500gK00000JCbtiQAD"
	caseObj, err := apiClient.GetCase(ctx, caseID)
	if err != nil {
		return nil, err //Failed to get case
	}
	return caseObj, nil
}
