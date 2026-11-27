package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type recordedRequest struct {
	method string
	path   string
	query  string
	body   []byte
}

// newRecordingClient returns a client whose every request is answered with the
// given JSON body, and the request it last made.
func newRecordingClient(t *testing.T, responseBody string) (*Client, *recordedRequest) {
	t.Helper()

	got := &recordedRequest{}
	c, err := NewClient(
		WithAPIKey("test-key"),
		WithURL("https://example.test"),
		WithTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			got.method = r.Method
			got.path = r.URL.Path
			got.query = r.URL.RawQuery
			if r.Body != nil {
				got.body, _ = io.ReadAll(r.Body)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(responseBody)),
				Header:     make(http.Header),
			}, nil
		})),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, got
}

func (r *recordedRequest) check(t *testing.T, method, path, query string) {
	t.Helper()
	if r.method != method {
		t.Errorf("method = %q, want %q", r.method, method)
	}
	if r.path != path {
		t.Errorf("path = %q, want %q", r.path, path)
	}
	if r.query != query {
		t.Errorf("query = %q, want %q", r.query, query)
	}
}

func (r *recordedRequest) checkBody(t *testing.T, want string) {
	t.Helper()
	var got, wantValue any
	if err := json.Unmarshal(r.body, &got); err != nil {
		t.Fatalf("request body %q is not JSON: %v", r.body, err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("want %q is not JSON: %v", want, err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(wantValue)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("request body = %s, want %s", gotJSON, wantJSON)
	}
}

func TestReadConversationDeprecated(t *testing.T) {
	c, got := newRecordingClient(t, `{"id":"c1","customer_id":"u1","channel":"web","status":"active","agent_is_active":true,"latest_intent":"refund","latest_handoff_target":"billing"}`)

	conv, err := c.ReadConversationDeprecated(context.Background(), "c1")
	if err != nil {
		t.Fatalf("ReadConversationDeprecated: %v", err)
	}
	got.check(t, http.MethodGet, "/conversations/c1", "")
	if conv.LatestIntent != "refund" || conv.LatestHandOffTarget != "billing" {
		t.Errorf("latest intent and hand-off target = %q, %q", conv.LatestIntent, conv.LatestHandOffTarget)
	}
}

func TestBatchCreateCustomerMemories(t *testing.T) {
	c, got := newRecordingClient(t, "")

	err := c.BatchCreateCustomerMemories(context.Background(), "cust-1234", BatchCreateMemoriesParams{
		Memories: []Memory{{
			ExternalID: "order_123",
			CustomType: "order",
			CreatedAt:  time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
			Data:       json.RawMessage(`{"status":"shipped"}`),
		}},
	})
	if err != nil {
		t.Fatalf("BatchCreateCustomerMemories: %v", err)
	}
	got.check(t, http.MethodPost, "/customers/cust-1234/memories", "")
	got.checkBody(t, `{"memories":[{"external_id":"order_123","custom_type":"order","created_at":"2026-07-01T10:00:00Z","data":{"status":"shipped"}}]}`)
}

func TestBatchCreateCustomerMemoriesV2(t *testing.T) {
	c, got := newRecordingClient(t, "")

	err := c.BatchCreateCustomerMemoriesV2(context.Background(), "cust-1234", BatchCreateMemoriesV2Params{
		Memories: []MemoryV2{{
			ExternalID: "order_123",
			CustomType: MemoryCustomType{Name: "order", Description: "Orders the customer placed"},
			CreatedAt:  time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
			Data:       json.RawMessage(`{"status":"shipped"}`),
		}},
	})
	if err != nil {
		t.Fatalf("BatchCreateCustomerMemoriesV2: %v", err)
	}
	got.check(t, http.MethodPost, "/customers/cust-1234/memories/v2", "")
	got.checkBody(t, `{"memories":[{"external_id":"order_123","custom_type":{"name":"order","description":"Orders the customer placed"},"created_at":"2026-07-01T10:00:00Z","data":{"status":"shipped"}}]}`)
}

func TestListCustomerMemories(t *testing.T) {
	c, got := newRecordingClient(t, `{"memories":[{"external_id":"order_123","custom_type":{"name":"order"},"created_at":"2026-07-01T10:00:00Z","data":{"status":"shipped"}}]}`)

	rsp, err := c.ListCustomerMemories(context.Background(), "cust-1234")
	if err != nil {
		t.Fatalf("ListCustomerMemories: %v", err)
	}
	got.check(t, http.MethodGet, "/customers/cust-1234/memories", "")
	if len(rsp.Memories) != 1 || rsp.Memories[0].CustomType.Name != "order" || string(rsp.Memories[0].Data) != `{"status":"shipped"}` {
		t.Errorf("memories = %+v", rsp.Memories)
	}
}

func TestFileUploadFlow(t *testing.T) {
	c, got := newRecordingClient(t, `{"upload_id":"up_1","upload_url":"https://storage.example.test/up_1","content_type":"image/png","headers":{"Content-Type":"image/png"},"expires_at":"2024-01-15T10:45:00Z","max_bytes":5242880}`)

	upload, err := c.BeginFileUpload(context.Background(), &BeginFileUploadParams{ContentType: "image/png"})
	if err != nil {
		t.Fatalf("BeginFileUpload: %v", err)
	}
	got.check(t, http.MethodPost, "/file-uploads", "")
	got.checkBody(t, `{"content_type":"image/png"}`)
	if upload.UploadID != "up_1" || upload.MaxBytes != 5242880 || upload.Headers["Content-Type"] != "image/png" {
		t.Errorf("upload = %+v", upload)
	}

	c, got = newRecordingClient(t, `{"id":"companyfile_1","ref":"id-card","file_name":"id-card.png","content_type":"image/png","size_bytes":482113,"created":"2024-01-15T10:30:00Z"}`)
	file, err := c.CreateFile(context.Background(), &CreateFileParams{UploadID: "up_1", FileName: "id-card.png"})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	got.check(t, http.MethodPost, "/files", "")
	got.checkBody(t, `{"upload_id":"up_1","file_name":"id-card.png"}`)
	if file.ID != "companyfile_1" || file.Ref != "id-card" || file.SizeBytes != 482113 {
		t.Errorf("file = %+v", file)
	}
}

func TestListFiles(t *testing.T) {
	c, got := newRecordingClient(t, `{"files":[{"id":"companyfile_1","ref":"id-card","file_name":"id-card.png","content_type":"image/png","size_bytes":1,"created":"2024-01-15T10:30:00Z"}],"pagination":{"next":"abc"}}`)

	rsp, err := c.ListFiles(context.Background(), &ListFilesParams{Cursor: "a b"})
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	got.check(t, http.MethodGet, "/files", "cursor=a+b")
	if len(rsp.Files) != 1 || rsp.Pagination == nil || *rsp.Pagination.Next != "abc" {
		t.Errorf("response = %+v", rsp)
	}

	if _, err := c.ListFiles(context.Background(), nil); err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	got.check(t, http.MethodGet, "/files", "")
}

func TestListIPAddresses(t *testing.T) {
	c, got := newRecordingClient(t, `{"api":["34.107.221.41/32"],"egress":["35.234.150.140/32"]}`)

	rsp, err := c.ListIPAddresses(context.Background())
	if err != nil {
		t.Fatalf("ListIPAddresses: %v", err)
	}
	got.check(t, http.MethodGet, "/ip-addresses", "")
	if len(rsp.API) != 1 || rsp.API[0] != "34.107.221.41/32" || len(rsp.Egress) != 1 {
		t.Errorf("response = %+v", rsp)
	}
}

func TestKnowledgeBases(t *testing.T) {
	const kb = `{"gradient_labs_id":"knowledgebase_1","id":"help-centre","name":"Help Centre","created":"2024-01-15T10:30:00Z","updated":"2024-01-15T11:00:00Z"}`

	t.Run("create", func(t *testing.T) {
		c, got := newRecordingClient(t, kb)
		rsp, err := c.CreateKnowledgeBase(context.Background(), &CreateKnowledgeBaseParams{ID: "help-centre", Name: "Help Centre"})
		if err != nil {
			t.Fatalf("CreateKnowledgeBase: %v", err)
		}
		got.check(t, http.MethodPost, "/knowledge-bases", "")
		got.checkBody(t, `{"id":"help-centre","name":"Help Centre"}`)
		if rsp.ID != "knowledgebase_1" || rsp.ExternalID != "help-centre" {
			t.Errorf("knowledge base = %+v", rsp)
		}
	})

	t.Run("list", func(t *testing.T) {
		c, got := newRecordingClient(t, `{"knowledge_bases":[`+kb+`]}`)
		rsp, err := c.ListKnowledgeBases(context.Background())
		if err != nil {
			t.Fatalf("ListKnowledgeBases: %v", err)
		}
		got.check(t, http.MethodGet, "/knowledge-bases", "")
		if len(rsp.KnowledgeBases) != 1 {
			t.Errorf("knowledge bases = %+v", rsp.KnowledgeBases)
		}
	})

	t.Run("read", func(t *testing.T) {
		c, got := newRecordingClient(t, kb)
		rsp, err := c.ReadKnowledgeBase(context.Background(), "help-centre")
		if err != nil {
			t.Fatalf("ReadKnowledgeBase: %v", err)
		}
		got.check(t, http.MethodGet, "/knowledge-bases/help-centre", "")
		if rsp.Name != "Help Centre" {
			t.Errorf("knowledge base = %+v", rsp)
		}
	})

	t.Run("update", func(t *testing.T) {
		c, got := newRecordingClient(t, kb)
		if _, err := c.UpdateKnowledgeBase(context.Background(), "help-centre", &UpdateKnowledgeBaseParams{Name: "Help Centre (EU)"}); err != nil {
			t.Fatalf("UpdateKnowledgeBase: %v", err)
		}
		got.check(t, http.MethodPost, "/knowledge-bases/help-centre", "")
		got.checkBody(t, `{"name":"Help Centre (EU)"}`)
	})
}

func TestNotes(t *testing.T) {
	const note = `{"gradient_labs_id":"note_1","id":"refunds","title":"Refunds","status":"live","last_modified_by":"user_1","created":"2024-01-15T10:30:00Z","updated":"2024-01-15T11:00:00Z","attributes":{"Locale":["UK"]}}`

	t.Run("list", func(t *testing.T) {
		c, got := newRecordingClient(t, `{"notes":[`+note+`],"pagination":{"next":"next-cursor"}}`)
		rsp, err := c.ListNotes(context.Background(), &ListNotesParams{Cursor: "cursor=="})
		if err != nil {
			t.Fatalf("ListNotes: %v", err)
		}
		got.check(t, http.MethodGet, "/notes", "cursor=cursor%3D%3D")
		if len(rsp.Notes) != 1 || rsp.Notes[0].ExternalID != "refunds" || rsp.Notes[0].Attributes["Locale"][0] != "UK" {
			t.Errorf("notes = %+v", rsp.Notes)
		}
		if rsp.Pagination == nil || rsp.Pagination.Next == nil || *rsp.Pagination.Next != "next-cursor" {
			t.Errorf("pagination = %+v", rsp.Pagination)
		}
	})

	t.Run("list without params", func(t *testing.T) {
		c, got := newRecordingClient(t, `{"notes":[],"pagination":{}}`)
		if _, err := c.ListNotes(context.Background(), nil); err != nil {
			t.Fatalf("ListNotes: %v", err)
		}
		got.check(t, http.MethodGet, "/notes", "")
	})

	t.Run("read", func(t *testing.T) {
		c, got := newRecordingClient(t, note)
		rsp, err := c.ReadNote(context.Background(), "refunds")
		if err != nil {
			t.Fatalf("ReadNote: %v", err)
		}
		got.check(t, http.MethodGet, "/notes/refunds", "")
		if rsp.ID != "note_1" || rsp.Status != NoteStatusLive {
			t.Errorf("note = %+v", rsp)
		}
	})

	t.Run("create sends attributes and knowledge bases", func(t *testing.T) {
		c, got := newRecordingClient(t, note)
		_, err := c.CreateNote(context.Background(), &CreateNoteParams{
			ID:               "refunds",
			Title:            "Refunds",
			Attributes:       map[string][]string{"Locale": {"UK"}},
			KnowledgeBaseIDs: []string{"help-centre"},
		})
		if err != nil {
			t.Fatalf("CreateNote: %v", err)
		}
		got.check(t, http.MethodPost, "/notes", "")
		got.checkBody(t, `{"id":"refunds","title":"Refunds","attributes":{"Locale":["UK"]},"knowledge_base_ids":["help-centre"]}`)
	})

	t.Run("update sends attributes and knowledge bases", func(t *testing.T) {
		c, got := newRecordingClient(t, note)
		_, err := c.UpdateNote(context.Background(), "refunds", &UpdateNoteParams{
			Title:            "Refunds",
			Attributes:       map[string][]string{"Locale": {"UK"}},
			KnowledgeBaseIDs: []string{"help-centre"},
		})
		if err != nil {
			t.Fatalf("UpdateNote: %v", err)
		}
		got.check(t, http.MethodPost, "/notes/refunds", "")
		got.checkBody(t, `{"title":"Refunds","attributes":{"Locale":["UK"]},"knowledge_base_ids":["help-centre"]}`)
	})
}

func TestUpsertArticleSendsPublicURLAndKnowledgeBases(t *testing.T) {
	c, got := newRecordingClient(t, "")

	err := c.UpsertArticle(context.Background(), &UpsertArticleParams{
		ID:                "a1",
		PublicationStatus: PublicationStatusExcluded,
		PublicURL:         "https://help.example.test/a1",
		KnowledgeBaseIDs:  []string{"help-centre"},
		Data:              json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("UpsertArticle: %v", err)
	}
	got.check(t, http.MethodPost, "/articles", "")

	var sent map[string]any
	if err := json.Unmarshal(got.body, &sent); err != nil {
		t.Fatal(err)
	}
	if sent["status"] != "excluded" || sent["public_url"] != "https://help.example.test/a1" {
		t.Errorf("body = %s", got.body)
	}
	if kbs, _ := sent["knowledge_base_ids"].([]any); len(kbs) != 1 || kbs[0] != "help-centre" {
		t.Errorf("knowledge_base_ids = %v", sent["knowledge_base_ids"])
	}
}

func TestUpsertArticleLeavesKnowledgeBasesOutByDefault(t *testing.T) {
	c, got := newRecordingClient(t, "")
	if err := c.UpsertArticle(context.Background(), &UpsertArticleParams{ID: "a1", Data: json.RawMessage(`{}`)}); err != nil {
		t.Fatalf("UpsertArticle: %v", err)
	}
	if strings.Contains(string(got.body), "knowledge_base_ids") {
		t.Errorf("body = %s", got.body)
	}
}

func TestUpsertArticleTopicSendsPublicURL(t *testing.T) {
	c, got := newRecordingClient(t, "")
	if err := c.UpsertArticleTopic(context.Background(), &UpsertArticleTopicParams{ID: "t1", PublicURL: "https://help.example.test/t1", Data: json.RawMessage(`{}`)}); err != nil {
		t.Fatalf("UpsertArticleTopic: %v", err)
	}
	var sent map[string]any
	if err := json.Unmarshal(got.body, &sent); err != nil {
		t.Fatal(err)
	}
	if sent["public_url"] != "https://help.example.test/t1" {
		t.Errorf("body = %s", got.body)
	}
}

func TestListTopicsDecodesTheWireFormat(t *testing.T) {
	c, got := newRecordingClient(t, `{"Topics":[{"Source":"public-api","ExternalID":"billing","Name":"Billing","Description":"Invoices","Visibility":"unknown","ParentExternalID":"account","Created":"2024-01-15T10:30:00Z","LastEdited":"2024-01-15T11:00:00Z","LastSeen":"2024-01-16T09:00:00Z","Data":"eyJhIjoxfQ==","PublicURL":"https://help.example.test/billing"}]}`)

	rsp, err := c.ListTopics(context.Background(), &ListTopicsParams{SupportPlatform: "public-api"})
	if err != nil {
		t.Fatalf("ListTopics: %v", err)
	}
	got.check(t, http.MethodGet, "/topics", "support_platform=public-api")
	if len(rsp.Topics) != 1 {
		t.Fatalf("topics = %+v", rsp.Topics)
	}
	topic := rsp.Topics[0]
	if topic.ExternalID != "billing" || topic.ParentExternalID != "account" || topic.PublicURL != "https://help.example.test/billing" {
		t.Errorf("topic = %+v", topic)
	}
	if topic.Visibility != VisibilityUnknown {
		t.Errorf("visibility = %q", topic.Visibility)
	}
	if string(topic.Data) != `{"a":1}` {
		t.Errorf("data = %q", topic.Data)
	}
	if topic.LastEdited.IsZero() || topic.LastSeen.IsZero() {
		t.Errorf("topic dates = %v, %v", topic.LastEdited, topic.LastSeen)
	}
}

func TestListProceduresQuery(t *testing.T) {
	c, got := newRecordingClient(t, `{"procedures":[],"pagination":{}}`)

	if _, err := c.ListProcedures(context.Background(), &ProcedureListParams{Cursor: "abc", Status: ProcedureStatusArchived}); err != nil {
		t.Fatalf("ListProcedures: %v", err)
	}
	got.check(t, http.MethodGet, "/procedures", "cursor=abc&status=archived")
}

func TestListProcedureVersionsDecodesTheWireFormat(t *testing.T) {
	c, _ := newRecordingClient(t, `{"Versions":[{"Name":"Refunds","Description":"d","Version":3,"Author":"user_1","Created":"2024-01-15T10:30:00Z","Gated":true,"GatedConfig":{"MaxDailyConversations":50},"Live":false}]}`)

	rsp, err := c.ListProcedureVersions(context.Background(), "proc_1")
	if err != nil {
		t.Fatalf("ListProcedureVersions: %v", err)
	}
	if len(rsp.Versions) != 1 {
		t.Fatalf("versions = %+v", rsp.Versions)
	}
	v := rsp.Versions[0]
	if v.Version != 3 || !v.Gated || v.GatedConfig == nil || v.GatedConfig.MaxDailyConversations != 50 {
		t.Errorf("version = %+v", v)
	}
}

func TestHandOffTargetRef(t *testing.T) {
	c, got := newRecordingClient(t, `{"targets":[{"id":"billing","name":"Billing team","ref":"billing-team"}]}`)

	rsp, err := c.ListHandOffTargets(context.Background())
	if err != nil {
		t.Fatalf("ListHandOffTargets: %v", err)
	}
	if rsp.Targets[0].Ref != "billing-team" {
		t.Errorf("ref = %q", rsp.Targets[0].Ref)
	}
	got.check(t, http.MethodGet, "/hand-off-targets", "")

	c, got = newRecordingClient(t, "")
	if err := c.UpsertHandOffTarget(context.Background(), &UpsertHandOffTargetParams{ID: "billing", Name: "Billing team", Ref: "billing-team"}); err != nil {
		t.Fatalf("UpsertHandOffTarget: %v", err)
	}
	got.checkBody(t, `{"id":"billing","name":"Billing team","ref":"billing-team"}`)
}

func TestReadLatestVoiceCallContextDecodesLiveHandoffURL(t *testing.T) {
	c, _ := newRecordingClient(t, `{"started_at":"2024-01-15T10:30:00Z","live_handoff_url":"https://app.example.test/live"}`)

	rsp, err := c.ReadLatestVoiceCallContext(context.Background(), "+447700900123", nil)
	if err != nil {
		t.Fatalf("ReadLatestVoiceCallContext: %v", err)
	}
	if rsp.LiveHandoffURL != "https://app.example.test/live" {
		t.Errorf("live handoff url = %q", rsp.LiveHandoffURL)
	}
}

func TestBackOfficeTaskCreateAlwaysSendsAgentID(t *testing.T) {
	c, got := newRecordingClient(t, `{"id":"t1","agent_id":"agent_1","status":"handed-off","hand_off_reason":"guardrail-violation","input":{},"created":"2024-01-15T10:30:00Z"}`)

	task, err := c.CreateBackOfficeTask(context.Background(), BackOfficeTaskCreateParams{ID: "t1", ProcedureID: "proc_1", Input: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatalf("CreateBackOfficeTask: %v", err)
	}
	if !strings.Contains(string(got.body), `"agent_id":""`) {
		t.Errorf("body = %s", got.body)
	}
	if task.Status != BackOfficeTaskStatusHandedOff || task.HandOffReason != BackOfficeTaskHandOffReasonGuardrailViolation {
		t.Errorf("status, reason = %q, %q", task.Status, task.HandOffReason)
	}
}

func TestResourceSourceDecodesTheContract(t *testing.T) {
	c, _ := newRecordingClient(t, `{"id":"resourcesrc_1","display_name":"Orders","description":"","source_type":"internal","available_refresh_strategies":["dynamic","static"],"available_scopes":["global","local"],"managed":{"enabled":true,"key":"intercom.contact"},"schema":{"raw":{},"attributes":[{"path":"$.id","name":"id","type":"string","cardinality":"one","is_root":true}]},"created":"2024-01-15T10:30:00Z","updated":"2024-01-15T10:30:00Z"}`)

	src, err := c.ReadResourceSource(context.Background(), "resourcesrc_1")
	if err != nil {
		t.Fatalf("ReadResourceSource: %v", err)
	}
	if src.Managed == nil || !src.Managed.Enabled || src.Managed.Key != "intercom.contact" {
		t.Errorf("managed = %+v", src.Managed)
	}
	if len(src.AvailableRefreshStrategies) != 2 || src.AvailableRefreshStrategies[0] != RefreshStrategyDynamic {
		t.Errorf("refresh strategies = %v", src.AvailableRefreshStrategies)
	}
	if len(src.AvailableScopes) != 2 || src.AvailableScopes[1] != ScopeLocal {
		t.Errorf("scopes = %v", src.AvailableScopes)
	}
	if src.Schema == nil || len(src.Schema.Attributes) != 1 || !src.Schema.Attributes[0].IsTopLevel {
		t.Errorf("schema = %+v", src.Schema)
	}
}

func TestResourceSourceAndTypeCreateSendBranch(t *testing.T) {
	c, got := newRecordingClient(t, `{}`)
	if _, err := c.CreateResourceSource(context.Background(), &ResourceSourceCreateParams{DisplayName: "Orders", SourceType: SourceTypeWebhook, BranchName: "copy-tools"}); err != nil {
		t.Fatalf("CreateResourceSource: %v", err)
	}
	if !strings.Contains(string(got.body), `"branch_name":"copy-tools"`) {
		t.Errorf("body = %s", got.body)
	}

	if _, err := c.CreateResourceType(context.Background(), &ResourceTypeCreateParams{DisplayName: "Orders", BranchID: "branch_1"}); err != nil {
		t.Fatalf("CreateResourceType: %v", err)
	}
	if !strings.Contains(string(got.body), `"branch_id":"branch_1"`) {
		t.Errorf("body = %s", got.body)
	}
}

func TestUpdateResourceSourceSendsSchema(t *testing.T) {
	c, got := newRecordingClient(t, `{}`)
	_, err := c.UpdateResourceSource(context.Background(), "resourcesrc_1", &ResourceSourceUpdateParams{
		Schema: &ResourceSchemaUpdate{Raw: json.RawMessage(`{}`)},
	})
	if err != nil {
		t.Fatalf("UpdateResourceSource: %v", err)
	}
	got.check(t, http.MethodPut, "/resource-sources/resourcesrc_1", "")
	if !strings.Contains(string(got.body), `"schema":{"raw":{}}`) {
		t.Errorf("body = %s", got.body)
	}
}

func TestCreateToolSendsTheContractFields(t *testing.T) {
	const response = `{"id":"tool_1","name":"Look up order","description":"d","parameters":[{"name":"file","description":"d","type":"attachment","allowed_sources":["llm","resource"],"transfer_mode":"temporary_url"}],"parameter_sets":[{"discriminator_parameter_name":"kind","discriminator_value":"refund","parameters":[]}],"draft":false,"async":{"start_execution_tool":{"http":{"method":"POST","url_template":"https://example.test","retry":{"max_attempts":3,"request_timeout_seconds":20}}},"cancel_execution_tool":{"webhook":{"name":"cancel"}},"timeout":30000000000}}`
	c, got := newRecordingClient(t, response)

	retries := 3
	tool, err := c.CreateTool(context.Background(), &CreateToolParams{
		Name:        "Look up order",
		Description: "d",
		BranchName:  "copy-tools",
		Parameters: []ToolParameter{{
			Name:           "file",
			Description:    "d",
			Type:           ParameterTypeAttachment,
			AllowedSources: []ParameterSource{ParameterSourceLLM, ParameterSourceResource},
			TransferMode:   TransferModeTemporaryURL,
		}},
		HTTP: &HTTPDefinition{Method: "POST", URLTemplate: "https://example.test", Retry: &HTTPRetryDefinition{MaxAttempts: &retries}},
	})
	if err != nil {
		t.Fatalf("CreateTool: %v", err)
	}
	got.check(t, http.MethodPost, "/tools", "")
	got.checkBody(t, `{"name":"Look up order","description":"d","branch_name":"copy-tools","parameters":[{"name":"file","description":"d","type":"attachment","allowed_sources":["llm","resource"],"transfer_mode":"temporary_url"}],"http":{"method":"POST","url_template":"https://example.test","retry":{"max_attempts":3}}}`)

	if len(tool.ParameterSets) != 1 || tool.ParameterSets[0].DiscriminatorValue != "refund" {
		t.Errorf("parameter sets = %+v", tool.ParameterSets)
	}
	if tool.Async == nil || tool.Async.CancelExecutionTool == nil || tool.Async.CancelExecutionTool.Webhook.Name != "cancel" {
		t.Errorf("async = %+v", tool.Async)
	}
	retry := tool.Async.StartExecutionTool.HTTP.Retry
	if retry == nil || *retry.MaxAttempts != 3 || *retry.RequestTimeoutSeconds != 20 {
		t.Errorf("retry = %+v", retry)
	}
	if tool.Parameters[0].TransferMode != TransferModeTemporaryURL {
		t.Errorf("transfer mode = %q", tool.Parameters[0].TransferMode)
	}
}

func TestUpdateToolKeepsTheIDOutOfTheBody(t *testing.T) {
	c, got := newRecordingClient(t, `{}`)
	_, err := c.UpdateTool(context.Background(), &UpdateToolParams{
		ID:          "tool_1",
		Description: "d",
		Async:       &AsyncDefinition{StartExecutionTool: ChildTool{Webhook: &ToolWebhookConfiguration{Name: "start"}}},
	})
	if err != nil {
		t.Fatalf("UpdateTool: %v", err)
	}
	got.check(t, http.MethodPut, "/tools/tool_1", "")
	if strings.Contains(string(got.body), `"id"`) || !strings.Contains(string(got.body), `"async"`) {
		t.Errorf("body = %s", got.body)
	}
}

func TestExecuteToolSendsToken(t *testing.T) {
	c, got := newRecordingClient(t, `{"id":"exec_1"}`)
	_, err := c.ExecuteTool(context.Background(), &ExecutionParams{ID: "tool_1", Arguments: []Argument{{Name: "order_id", Value: "1"}}, Token: "tok"})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	got.check(t, http.MethodPost, "/tools/tool_1/execute", "")
	got.checkBody(t, `{"arguments":[{"name":"order_id","value":"1"}],"token":"tok"}`)
}

func TestSecretRefreshMechanismSendsRetry(t *testing.T) {
	c, got := newRecordingClient(t, `{"name":"s","created":"2024-01-15T10:30:00Z","updated":"2024-01-15T10:30:00Z"}`)
	seconds := 2
	_, err := c.WriteSecret(context.Background(), &WriteSecretParams{
		Name:  "s",
		Value: "v",
		RefreshMechanismHTTP: &RefreshMechanismHTTP{
			RequestDefinition: SecretRefreshRequest{Method: "POST", URLTemplate: "https://example.test", Retry: &SecretRefreshRetry{RequestTimeoutSeconds: &seconds}},
			ResponseParamName: "access_token",
		},
	})
	if err != nil {
		t.Fatalf("WriteSecret: %v", err)
	}
	if !strings.Contains(string(got.body), `"retry":{"request_timeout_seconds":2}`) {
		t.Errorf("body = %s", got.body)
	}
}

func TestReadConversationWithoutOptions(t *testing.T) {
	c, got := newRecordingClient(t, `{"id":"c1"}`)
	if _, err := c.ReadConversation(context.Background(), "c1", nil); err != nil {
		t.Fatal(err)
	}
	got.check(t, http.MethodGet, "/conversations/c1/read", "")
}

func TestAddMessageWithDefaultTimestamp(t *testing.T) {
	c, got := newRecordingClient(t, `{"id":"m1"}`)
	_, err := c.AddMessage(context.Background(), "c1", AddMessageParams{ID: "m1", Body: "Hello", ParticipantType: ParticipantTypeCustomer})
	if err != nil {
		t.Fatal(err)
	}
	got.checkBody(t, `{"id":"m1","body":"Hello","participant_type":"Customer"}`)
}

func TestSetProcedureLimitReadsReturnedProcedure(t *testing.T) {
	c, got := newRecordingClient(t, `{"procedure":{"id":"proc_1"}}`)
	proc, err := c.SetProcedureLimit(context.Background(), "proc_1", &ProcedureLimitParams{})
	if err != nil {
		t.Fatal(err)
	}
	got.check(t, http.MethodPost, "/procedure/proc_1/limit", "")
	if proc.ID != "proc_1" {
		t.Fatalf("procedure ID = %q", proc.ID)
	}
}

func TestToolParameterLists(t *testing.T) {
	c, got := newRecordingClient(t, `{}`)
	_, err := c.CreateTool(context.Background(), &CreateToolParams{Name: "lookup", Description: "Find an order", Draft: true, BranchName: "orders"})
	if err != nil {
		t.Fatal(err)
	}
	got.checkBody(t, `{"name":"lookup","description":"Find an order","parameters":[],"draft":true,"branch_name":"orders"}`)
	sets := []ToolParameterSet{}
	_, err = c.UpdateTool(context.Background(), &UpdateToolParams{ID: "tool_1", ParameterSets: &sets})
	if err != nil {
		t.Fatal(err)
	}
	got.checkBody(t, `{"description":"","parameter_sets":[]}`)
}

func TestListToolsReturnsMCPConfiguration(t *testing.T) {
	c, _ := newRecordingClient(t, `{"tools":[{"id":"tool_1","name":"lookup","description":"Look up an order","parameters":[],"mcp":{"server_id":"mcpserver_1","external_tool_name":"lookup_order"}}]}`)
	tools, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].MCP == nil {
		t.Fatalf("expected one MCP tool, got %+v", tools)
	}
	if got := tools[0].MCP; got.ServerID != "mcpserver_1" || got.ExternalToolName != "lookup_order" {
		t.Errorf("mcp = %+v", got)
	}
}
