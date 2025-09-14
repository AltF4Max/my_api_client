package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"my-api-client/examples"
	"my-api-client/internal/client"

	"gopkg.in/yaml.v3"
)

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
		ToEmail      string `yaml:"to_email"`
		LogFile      string `yaml:"log_file"`
		LogLevel     string `yaml:"log_level"`
	} `yaml:"salesforce"`
}

func main() {
	// Loading configuration from file in config folder
	authConfig, err := loadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// Creating a client
	apiClient := client.NewAPIClient(authConfig)
	defer apiClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. CreateCase
	createdCase, err := examples.ExamplesCreateCase(ctx, apiClient)
	if err != nil {
		log.Printf("Failed to create case: %v", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Created case with ID: %s\n", createdCase.ID)

	// 2. GetCase
	caseObj, err := examples.ExamplesGetCase(ctx, apiClient)
	if err != nil {
		log.Printf("Failed to get case: %v", err)
		os.Exit(1)
	}
	fmt.Printf("📋 Case Subject: %s\n", caseObj.Subject)
	fmt.Printf("📊 Case Status: %s\n", caseObj.Status)
	fmt.Printf("🎯 Case Priority: %s\n", caseObj.Priority)

	// 3. Query
	result, err := examples.ExamplesQuery(ctx, apiClient)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		os.Exit(1)
	}
	fmt.Printf("🔍 Found %d cases\n", result.TotalSize)
	for _, record := range result.Records {
		fmt.Printf("Record: %+v\n", record)
	}

	// 4. CreateAttachment
	result2, err := examples.ExamplesCreateAttachment(ctx, apiClient)
	if err != nil {
		log.Printf("Failed to create attachment: %v", err)
		os.Exit(1)
	}
	if success, ok := result2["success"].(bool); ok && success {
		fmt.Printf("📎 Attachment created successfully\n")
		if data, ok := result2["data"].(map[string]interface{}); ok {
			if id, ok := data["id"].(string); ok {
				fmt.Printf("Attachment ID: %s\n", id)
			}
		}
	} else {
		fmt.Printf("❌ Attachment creation failed: %+v\n", result2)
	}

	// 5. EmailMessage
	toAddress := config.Salesforce.ToEmail
	result3, err := examples.ExamplesEmailMessage(ctx, apiClient, toAddress)
	if err != nil {
		log.Printf("Error creating email message: %v", err)
		os.Exit(1)
	}
	fmt.Printf("✉️  Email message created: %+v\n", result3)

	fmt.Println("🎉 All operations completed successfully!")
}

func loadConfig(filename string) (*client.AuthConfig, error) {
	// Checking the existence of a file
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", filename)
	}

	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %v", err)
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
		LogFile:      config.Salesforce.LogFile,
		LogLevel:     config.Salesforce.LogLevel,
	}

	return authConfig, nil
}
