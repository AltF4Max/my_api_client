package examples

import (
	"context"
	"fmt"
	"log"
	"my-api-client/internal/client"
	"os"
)

func ExamplesCreateAttachment(ctx context.Context, apiClient *client.APIClient) (map[string]interface{}, error) {
	// Setting up Case ID
	apiClient.SetCaseID("500gK00000JCbtiQAD")

	// Path to an existing PDF file
	filePath := "C:\\Users\\Max\\Desktop\\my-api-client\\path\\to\\real_document.pdf"

	// Check that the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Fatalf("The file does not exist: %s\nPlease check the path and existence of the file.", filePath)
	}

	// Check that this is a file (not a directory)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Fatalf("Error getting file information: %v", err)
	}
	if fileInfo.IsDir() {
		log.Fatalf("The path specified leads to a directory, not a file: %s", filePath)
	}

	fmt.Printf("File used: %s\n", filePath)
	fmt.Printf("File size: %d byte\n", fileInfo.Size())

	result, err := apiClient.CreateAttachment(ctx, filePath)
	if err != nil {
		return nil, err //Failed to create attachment
	}
	return result, nil
}
