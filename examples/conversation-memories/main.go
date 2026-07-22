// This example demonstrates bulk-uploading memories scoped to a conversation,
// which the AI agent can then search over on demand.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

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

	result, err := client.BulkUploadConversationMemories(ctx, "conversation-1234", glabs.BulkUploadMemoriesParams{
		IdempotencyKey: "upload-2024-01-01",
		Memories: []json.RawMessage{
			json.RawMessage(`{"order_id": "A1", "status": "shipped", "created_at": "2024-01-01T10:00:00Z"}`),
			json.RawMessage(`{"order_id": "A2", "status": "delivered", "created_at": "2024-01-02T10:00:00Z"}`),
		},
		CreatedAtKeys: []string{"created_at"},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Upload %s inserted %d memories\n", result.UploadID, result.MemoriesInserted)
}
