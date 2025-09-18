package examples

import (
	"context"

	"github.com/AltF4Max/sf_api_client/internal/client"
)

func ExamplesCreateCase(ctx context.Context, apiClient *client.APIClient) (*client.Case, error) {
	// Create a new case with full data
	newCase := &client.Case{
		Status:        "New",
		Origin:        "Support",
		AccountId:     "", // Fill in if you have an Account ID
		ContactId:     "", // Fill in if you have Contact ID
		Subject:       "Test Case from Go",
		SuppliedName:  "Test User",        // Username
		SuppliedEmail: "test@example.com", // User email
		SuppliedPhone: "+1234567890",      // Telephone
		Description:   "This case was created from Go client with full details",
		Product:       "Advanced Subscription", // Product
		Type:          "Case",
		//RecordTypeId:     "01250000000DfSU",
		//SuppliedCountry:  "US",                    // Country code
		//IPAddress:        "192.168.1.1",           // IP address
		//Severity:         "Medium",                // The severity of the problem
		//OperatingSystem:  "CentOS Linux",          // OS
		//WebQueueEmail:    "sf-all@cyber.com",
		//WebURL:           "oem.cyber.com",
	}

	// Headers for creating a case
	caseHeaders := client.CaseHeaders{
		SforceAssignmentRuleHeader: "assignmentRuleId=01Q50000000A9r2EAC",
		SforceEmailHeader:          "triggerAutoResponseEmail=true, triggerUserEmail=true",
	}

	createdCase, err := apiClient.CreateCase(ctx, newCase, caseHeaders)
	if err != nil {
		return nil, err //Failed to create case
	}
	return createdCase, nil

}
