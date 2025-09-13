package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"my-api-client/internal/client"

	"gopkg.in/yaml.v3"
)

func loadConfig(filename string) (*client.AuthConfig, error) {
	// Checking the existence of a file
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", filename)
	}

	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %v", err)
	}

	// Structure for parsing given the root element salesforce
	var config struct {
		Salesforce struct {
			ClientID     string `yaml:"client_id"`
			ClientSecret string `yaml:"client_secret"`
			RefreshToken string `yaml:"refresh_token"`
			Username     string `yaml:"username"`
			Password     string `yaml:"password"`
			LoginURL     string `yaml:"login_url"`
			GrantType    string `yaml:"grant_type"`
			Debug        bool   `yaml:"debug"`
		} `yaml:"salesforce"`
	}

	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("YAML parse error: %v", err)
	}

	// Convert to client.AuthConfig
	authConfig := &client.AuthConfig{
		ClientID:     config.Salesforce.ClientID,
		ClientSecret: config.Salesforce.ClientSecret,
		RefreshToken: config.Salesforce.RefreshToken,
		Username:     config.Salesforce.Username,
		Password:     config.Salesforce.Password,
		LoginURL:     config.Salesforce.LoginURL,
		GrantType:    config.Salesforce.GrantType,
		Debug:        config.Salesforce.Debug,
	}

	return authConfig, nil
}
func main() {
	// Loading configuration from file in config folder
	authConfig, err := loadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	// Creating a client
	apiClient := client.NewAPIClient(authConfig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a new case with full data
	newCase := &client.Case{
		Status:        "New",
		Origin:        "OEM Portal",
		AccountId:     "", // Fill in if you have an Account ID
		ContactId:     "", // Fill in if you have Contact ID
		Subject:       "Test Case from Go",
		SuppliedName:  "Test User",        // Username
		SuppliedEmail: "test@example.com", // User email
		SuppliedPhone: "+1234567890",      // Telephone
		Description:   "This case was created from Go client with full details",
		Product:       "Acronis Cyber Protect", // Product
		Type:          "Case",
		// RecordTypeId:     "01250000000DfSU",
		//SuppliedCountry: "US",          // Country code
		//IPAddress:   "192.168.1.1", // IP address
		//Severity: "Medium",                // The severity of the problem
		//OperatingSystem: "CentOS Linux",          // OS
		//WebQueueEmail: "sf-all@acronis.com",
		//WebURL: "oem.acronis.com",
	}

	// Headers for creating a case
	caseHeaders := client.CaseHeaders{
		SforceAssignmentRuleHeader: "assignmentRuleId=01Q50000000A9r2EAC",
		SforceEmailHeader:          "triggerAutoResponseEmail=true, triggerUserEmail=true",
	}

	// Create a case with headers
	createdCase, err := apiClient.CreateCase(ctx, newCase, caseHeaders)
	if err != nil {
		log.Fatalf("Failed to create case: %v", err)
	}

	fmt.Printf("Created case with ID: %s\n", createdCase.ID)

	caseID := "500gK00000JCbtiQAD"
	caseObj, err := apiClient.GetCase(ctx, caseID)
	if err != nil {
		log.Fatalf("Failed to get case: %v", err)
	}
	fmt.Printf("Case Subject: %s\n", caseObj.Subject)
	fmt.Printf("Case Status: %s\n", caseObj.Status)
	fmt.Printf("Case Priority: %s\n", caseObj.Priority)
	// Executing a SOQL query
	soql := "SELECT Id, Subject, Status FROM Case LIMIT 5"
	result, err := apiClient.Query(ctx, soql)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}

	fmt.Printf("Found %d cases\n", result.TotalSize)
	for _, record := range result.Records {
		fmt.Printf("Record: %+v\n", record)
	}

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

	result2, err := apiClient.CreateAttachment(ctx, filePath)
	if err != nil {
		log.Fatalf("Failed to create attachment: %v", err)
	}

	if success, ok := result2["success"].(bool); ok && success {
		fmt.Printf("Attachment created successfully\n")
		if data, ok := result2["data"].(map[string]interface{}); ok {
			if id, ok := data["id"].(string); ok {
				fmt.Printf("Attachment ID: %s\n", id)
			}
		}
	} else {
		fmt.Printf("Attachment creation failed: %+v\n", result2)
	}

	// Creating parameters
	params := client.EmailMessageParams{
		CaseId:   "500gK00000JCbtiQAD", // ID case example
		From:     "user@example.com",
		Subject:  "Test Email Subject",
		TextBody: "This is the body of the test email",
		// To and Status will be set by default
	}

	// Calling a Method
	result3, err := apiClient.EmailMessage(context.Background(), params)
	if err != nil {
		log.Fatalf("Error creating email message: %v", err)
	}

	fmt.Printf("Email message created: %+v\n", result3)

}
