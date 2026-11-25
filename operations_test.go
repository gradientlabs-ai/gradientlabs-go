package client

import (
	"reflect"
	"testing"
)

// contractOperations maps each operation id in the public API contract, as
// recorded in PUBLIC_API_VERSION, to the Client method that implements it. The
// contract's Spec operation (GET /spec.json) is deliberately not exposed.
var contractOperations = map[string]string{
	"ArticleDelete":                     "DeleteArticle",
	"ArticleSetUsageStatus":             "SetArticleUsageStatus",
	"ArticleTopicUpsert":                "UpsertArticleTopic",
	"ArticleUpsert":                     "UpsertArticle",
	"BackOfficeTaskCreate":              "CreateBackOfficeTask",
	"BackOfficeTaskDelete":              "DeleteBackOfficeTask",
	"BackOfficeTaskRead":                "ReadBackOfficeTask",
	"ConversationAddMessage":            "AddMessage",
	"ConversationAssign":                "AssignConversation",
	"ConversationCancel":                "CancelConversation",
	"ConversationDelete":                "DeleteConversation",
	"ConversationEvent":                 "AddConversationEvent",
	"ConversationFinish":                "FinishConversation",
	"ConversationRate":                  "RateConversation",
	"ConversationRead":                  "ReadConversation",
	"ConversationReadDeprecated":        "ReadConversationDeprecated",
	"ConversationResume":                "ResumeConversation",
	"ConversationReturnAsyncToolResult": "ReturnAsyncToolResult",
	"ConversationStart":                 "StartConversation",
	"CustomerMemoriesBatchCreate":       "BatchCreateCustomerMemories",
	"CustomerMemoriesBatchCreateV2":     "BatchCreateCustomerMemoriesV2",
	"CustomerMemoriesList":              "ListCustomerMemories",
	"FileCreate":                        "CreateFile",
	"FileList":                          "ListFiles",
	"FileUploadBegin":                   "BeginFileUpload",
	"GetDefaultHandOffTarget":           "GetDefaultHandOffTarget",
	"HandOffTargetDelete":               "DeleteHandOffTarget",
	"HandOffTargetUpsert":               "UpsertHandOffTarget",
	"IPAddressList":                     "ListIPAddresses",
	"KnowledgeBaseCreate":               "CreateKnowledgeBase",
	"KnowledgeBaseList":                 "ListKnowledgeBases",
	"KnowledgeBaseRead":                 "ReadKnowledgeBase",
	"KnowledgeBaseUpdate":               "UpdateKnowledgeBase",
	"ListHandOffTargets":                "ListHandOffTargets",
	"ListProcedures":                    "ListProcedures",
	"ListTopics":                        "ListTopics",
	"ListTrafficGroups":                 "ListTrafficGroups",
	"NoteCreate":                        "CreateNote",
	"NoteDelete":                        "DeleteNote",
	"NoteList":                          "ListNotes",
	"NoteRead":                          "ReadNote",
	"NoteSetStatus":                     "SetNoteStatus",
	"NoteUpdate":                        "UpdateNote",
	"OutboundChatConversationStart":     "StartOutboundChatConversation",
	"OutboundEmailConversationStart":    "StartOutboundEmailConversation",
	"OutboundPhoneConversationStart":    "StartOutboundPhoneConversation",
	"ProcedureListVersions":             "ListProcedureVersions",
	"ProcedureSetGatedVersion":          "SetProcedureGatedVersion",
	"ProcedureSetLiveVersion":           "SetProcedureLiveVersion",
	"ProcedureUnsetGatedVersion":        "UnsetProcedureGatedVersion",
	"ProcedureUnsetLiveVersion":         "UnsetProcedureLiveVersion",
	"ReadProcedure":                     "ReadProcedure",
	"ReadTopic":                         "ReadTopic",
	"ResourceSourceCreate":              "CreateResourceSource",
	"ResourceSourceDelete":              "DeleteResourceSource",
	"ResourceSourceList":                "ListResourceSources",
	"ResourceSourceRead":                "ReadResourceSource",
	"ResourceSourceUpdate":              "UpdateResourceSource",
	"ResourceTypeCreate":                "CreateResourceType",
	"ResourceTypeDelete":                "DeleteResourceType",
	"ResourceTypeList":                  "ListResourceTypes",
	"ResourceTypeRead":                  "ReadResourceType",
	"ResourceTypeUpdate":                "UpdateResourceType",
	"SecretList":                        "ListSecrets",
	"SecretRevoke":                      "RevokeSecret",
	"SecretWrite":                       "WriteSecret",
	"SetDefaultHandOffTarget":           "SetDefaultHandOffTarget",
	"SetProcedureLimit":                 "SetProcedureLimit",
	"TerminologySubstitutionCreate":     "CreateTerminologySubstitution",
	"TerminologySubstitutionDelete":     "DeleteTerminologySubstitution",
	"TerminologySubstitutionList":       "ListTerminologySubstitutions",
	"TerminologySubstitutionRead":       "ReadTerminologySubstitution",
	"TerminologySubstitutionUpdate":     "UpdateTerminologySubstitution",
	"ToolCreate":                        "CreateTool",
	"ToolDelete":                        "DeleteTool",
	"ToolExecute":                       "ExecuteTool",
	"ToolList":                          "ListTools",
	"ToolRead":                          "ReadTool",
	"ToolUpdate":                        "UpdateTool",
	"TrafficGroupCreate":                "CreateTrafficGroup",
	"TrafficGroupDelete":                "DeleteTrafficGroup",
	"TrafficGroupExclusionCreate":       "CreateTrafficGroupExclusion",
	"TrafficGroupExclusionDelete":       "DeleteTrafficGroupExclusion",
	"TrafficGroupTargetCreate":          "CreateTrafficGroupTarget",
	"TrafficGroupTargetDelete":          "DeleteTrafficGroupTarget",
	"TrafficGroupUpdate":                "UpdateTrafficGroup",
	"UpdateSchemaByExamples":            "UpdateResourceSourceSchemaByExamples",
	"VoiceCallContextRead":              "ReadLatestVoiceCallContext",
}

func TestContractOperationsAreImplemented(t *testing.T) {
	clientType := reflect.TypeOf(&Client{})
	for operationID, method := range contractOperations {
		if _, ok := clientType.MethodByName(method); !ok {
			t.Errorf("operation %s: Client has no method %s", operationID, method)
		}
	}
}

func TestSpecOperationIsNotExposed(t *testing.T) {
	if _, ok := contractOperations["Spec"]; ok {
		t.Error("the Spec operation (GET /spec.json) must not be exposed")
	}
}
