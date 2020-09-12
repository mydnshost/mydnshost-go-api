package mydnshost_go_api_test

import (
	"context"
	mydnshost "github.com/mydnshost/mydnshost-go-api"
	"log"
)

const (
	userName = "user@example.com"
	apiKey   = "THIS-IS-A-DEMO-API-KEY"
)

// This example shows basic usage of the package: creating a client with appropriate credentials,
// retrieving a list of domains, and creating new records.
func Example_basic() {
	// Create a new client using API key authentication. For domain-specific keys, DomainKeyAuthenticator can be
	// used instead. Other authentication methods are not directly supported but can be implemented by conforming to
	// the ClientAuthenticator interface.
	client := &mydnshost.Client{
		Authenticator: &mydnshost.ApiKeyAuthenticator{
			User: userName,
			Key:  apiKey,
		},
	}

	// List all domains accessible using the API key, and the access level we have to them.
	domains, err := client.Domains(context.Background())
	if err != nil {
		log.Fatalf("Unable to list domains: %v", err)
	}

	for domain := range domains {
		log.Printf("Domain %s has access level %s\n", domain, domains[domain])
	}

	// Create a new record for one of our domains. ModifyRecords can also delete and update existing records by
	// using the DeleteRecord and ModifyRecord methods. Multiple operations can be performed in one API call.
	records, err := client.ModifyRecords(
		context.Background(),
		"example.com",
		mydnshost.CreateRecord(mydnshost.Record{
			Name:    "test",
			Type:    "A",
			Content: "0.0.0.0",
			TTL:     84600,
		}),
	)

	if err != nil {
		log.Fatalf("Unable to create record: %v", err)
	}

	for i := range records.Changed {
		r := records.Changed[i]
		log.Printf("Record %d (%s record for %s): updated = %t, deleted = %t", r.Id, r.Type, r.Name, r.Updated, r.Deleted)
	}
}
