// This example demonstrates batch-creating memories scoped to a customer,
// which the AI agent can then search over on demand.
//
// The call is asynchronous: it returns as soon as the batch job is accepted. It
// returns a 409 error if a batch is already being created for the same customer.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	glabs "github.com/gradientlabs-ai/gradientlabs-go"
)

func main() {
	opts := []glabs.Option{
		glabs.WithAPIKey(os.Getenv("GLABS_API_KEY")),
	}
	if baseURL := os.Getenv("GLABS_BASE_URL"); baseURL != "" {
		opts = append(opts, glabs.WithURL(baseURL))
	}

	client, err := glabs.NewClient(opts...)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	err = client.BatchCreateCustomerMemories(ctx, "customer-1234", glabs.BatchCreateMemoriesParams{
		Memories: []glabs.Memory{
			{
				ExternalID: "order_A1",
				CustomType: "order",
				CreatedAt:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				Data:       json.RawMessage(`{"order_id": "A1", "status": "shipped"}`),
			},
			{
				ExternalID: "order_A2",
				CustomType: "order",
				CreatedAt:  time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
				Data:       json.RawMessage(`{"order_id": "A2", "status": "delivered"}`),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("memory batch accepted")
}
