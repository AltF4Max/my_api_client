package examples

import (
	"context"
	"my-api-client/internal/client"
)

func ExamplesQuery(ctx context.Context, apiClient *client.APIClient) (*client.QueryResponse, error) {
	soql := "SELECT Id, Subject, Status FROM Case LIMIT 5"
	result, err := apiClient.Query(ctx, soql)
	if err != nil {
		return nil, err //Failed to execute query
	}
	return result, nil
}
