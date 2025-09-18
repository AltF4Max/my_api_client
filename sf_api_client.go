package my_api_client

import (
	"fmt"
	"os"

	"github.com/AltF4Max/sf_api_client/internal/client"

	"gopkg.in/yaml.v3"
)

type Case = client.Case
type CaseHeaders = client.CaseHeaders
type EmailMessageParams = client.EmailMessageParams
type APIClient = client.APIClient
type ErrorResponse = client.ErrorResponse

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

func GetToEmail() (string, error) {
	authConfig, err := loadConfig("sf_api_client/config/config.yaml")
	if err != nil {
		return "", err
	}
	return authConfig.ToEmail, nil
}

func NewAPIClientMax() (*client.APIClient, error) {
	// Loading configuration from file in config folder
	authConfig, err := loadConfig("sf_api_client/config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}
	// Creating a client
	return client.NewAPIClient(authConfig), nil
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
		ToEmail:      config.Salesforce.ToEmail,
		LogFile:      config.Salesforce.LogFile,
		LogLevel:     config.Salesforce.LogLevel,
	}

	return authConfig, nil
}
