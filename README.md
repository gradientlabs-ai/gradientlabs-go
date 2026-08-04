# Gradient Labs Go
[![Go Reference](https://pkg.go.dev/badge/github.com/gradientlabs-ai/gradientlabs-go.svg)](https://pkg.go.dev/github.com/gradientlabs-ai/gradientlabs-go)

Go bindings for the [Gradient Labs API](https://api-docs.gradient-labs.ai).

## Requirements

- Go 1.20 or later

## Installation

```bash
go get github.com/gradientlabs-ai/gradientlabs-go
```

## Documentation

- [API Documentation](https://api-docs.gradient-labs.ai)
- [Go Package Reference](https://pkg.go.dev/github.com/gradientlabs-ai/gradientlabs-go)

## Example Usage

### Starting a Conversation

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    glabs "github.com/gradientlabs-ai/gradientlabs-go"
)

func main() {
    // Create a new client with your API key
    client, err := glabs.NewClient(
        glabs.WithAPIKey(os.Getenv("GLABS_API_KEY")),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Start a new conversation
    conv, err := client.StartConversation(ctx, glabs.StartConversationParams{
        ID:         "conversation-1234",
        CustomerID: "user-1234",
        Channel:    glabs.ChannelWeb,
        Resources: map[string]any{
            "user_profile": map[string]any{
                "name":         "Jane Doe",
                "subscription": "premium",
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Started conversation: %s\n", conv.ID)

    // Add a message to the conversation
    msg, err := client.AddMessage(ctx, conv.ID, glabs.AddMessageParams{
        ID:              "message-1234",
        Body:            "Hello! I need some help.",
        ParticipantID:   "user-1234",
        ParticipantType: glabs.ParticipantTypeCustomer,
        Created:         time.Now(),
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Added message: %s\n", msg.ID)

    // Assign the conversation to an AI agent
    err = client.AssignConversation(ctx, conv.ID, &glabs.AssignmentParams{
        AssigneeType: glabs.ParticipantTypeAIAgent,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Conversation assigned to AI agent")
}
```

### Starting an Outbound Conversation

Outbound conversations have one method per channel:

```go
rsp, err := client.StartOutboundPhoneConversation(ctx, glabs.StartOutboundPhoneConversationParams{
    CustomerID:      "user-1234",
    ProcedureID:     "procedure-1234",
    ToPhoneNumber:   "+14155551234",
    FromPhoneNumber: "+14155550000",
})
```

```go
rsp, err := client.StartOutboundEmailConversation(ctx, glabs.StartOutboundEmailConversationParams{
    CustomerID:      "user-1234",
    ProcedureID:     "procedure-1234",
    SupportPlatform: glabs.SupportPlatformZendesk,
    CustomerSupportPlatformIdentifiers: []*glabs.CustomerSupportPlatformIdentifier{{
        SupportPlatform: glabs.SupportPlatformZendesk,
        Type:            glabs.CustomerSupportPlatformIdentifierTypeZendeskSupportUser,
        Value:           "987654321",
    }},
})
```

`StartOutboundChatConversation` takes the same parameters as the email variant,
minus `Subject`.

#### Migrating from `StartOutboundConversation`

`StartOutboundConversation` has been removed. Pick the method matching the
channel you were passing:

| Old `Channel`  | New method                        |
| -------------- | --------------------------------- |
| `ChannelVoice` | `StartOutboundPhoneConversation`  |
| `ChannelEmail` | `StartOutboundEmailConversation`  |
| `ChannelWeb`   | `StartOutboundChatConversation`   |

- `CustomerSource` has no replacement, and `CustomerID` is now always your own
  customer ID. Identifiers for third-party support platforms go in
  `CustomerSupportPlatformIdentifiers`, keyed by platform — Zendesk requires the
  `zendesk_support_user` type, and Salesforce the `salesforce_contact_id` type.
- `SupportPlatform` is required for chat and email; the highest-priority
  connected platform is no longer selected for you. Phone has no support
  platform: the call is always placed over voice.

For more examples, see the [examples](./examples) directory.
