package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danishjsheikh/swagger-mcp/app/models"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestExtractSchemaName(t *testing.T) {
	cases := []struct {
		ref, schemaType, want string
	}{
		{"#/components/schemas/User", "object", "User"},
		{"", "string", "string"},
	}
	for _, c := range cases {
		got := ExtractSchemaName(c.ref, c.schemaType)
		if got != c.want {
			t.Errorf("ExtractSchemaName(%q, %q) = %q, want %q", c.ref, c.schemaType, got, c.want)
		}
	}
}

func TestCompileRegexes(t *testing.T) {
	regexes := compileRegexes("/api/.*, /test/")
	if len(regexes) != 2 {
		t.Errorf("Expected 2 regexes, got %d", len(regexes))
	}
}

func TestShouldIncludePath(t *testing.T) {
	include := compileRegexes("/api/.*")
	exclude := compileRegexes("/api/private")
	if !shouldIncludePath("/api/test", include, exclude) {
		t.Error("Expected /api/test to be included")
	}
	if shouldIncludePath("/api/private", include, exclude) {
		t.Error("Expected /api/private to be excluded")
	}
}

func TestShouldIncludeMethod(t *testing.T) {
	if !shouldIncludeMethod("GET", []string{}, []string{}) {
		t.Error("Expected GET to be included by default")
	}
	if !shouldIncludeMethod("POST", []string{"POST"}, []string{}) {
		t.Error("Expected POST to be included")
	}
	if shouldIncludeMethod("DELETE", []string{"GET"}, []string{}) {
		t.Error("Expected DELETE to be excluded")
	}
	if shouldIncludeMethod("GET", []string{}, []string{"GET"}) {
		t.Error("Expected GET to be excluded by exclude list")
	}
}

func TestSetRequestSecurity_Basic(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	setRequestSecurity(req, "basic", "user:pass", "", "")
	if req.Header.Get("Authorization") == "" {
		t.Error("Expected Authorization header for basic auth")
	}
}

func TestSetRequestSecurity_Bearer(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	setRequestSecurity(req, "bearer", "", "", "token123")
	if req.Header.Get("Authorization") != "Bearer token123" {
		t.Error("Expected Bearer token in Authorization header")
	}
}

func TestSetRequestSecurity_ApiKey(t *testing.T) {
	// Use httptest.Server to inspect the actual outgoing request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Debug: print all headers
		t.Logf("Incoming headers: %v", r.Header)
		// Debug: print full URL
		t.Logf("Incoming URL: %s", r.URL.String())
		// Debug: print all cookies
		t.Logf("Incoming cookies: %v", r.Cookies())

		// Check header
		if got := r.Header.Get("X-API-KEY"); got != "abc" {
			t.Errorf("Expected X-API-KEY header to be 'abc', got '%s'", got)
		}
		// Check query param
		if got := r.URL.Query().Get("foo"); got != "bar2" {
			t.Errorf("Expected foo query param to be 'bar2', got '%s'", got)
		}
		// Check cookies
		found := false
		for _, c := range r.Cookies() {
			if c.Name == "sid" && c.Value == "ccc" {
				found = true
			}
		}
		if !found {
			t.Error("Expected sid cookie to be set")
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"?foo=bar", nil)
	setRequestSecurity(req, "apiKey", "header:X-API-KEY=abc,query:foo=bar2,cookie:sid=ccc", "", "")

	// Debug: print request before sending
	t.Logf("Outgoing request headers: %v", req.Header)
	t.Logf("Outgoing request URL: %s", req.URL.String())
	t.Logf("Outgoing request cookies: %v", req.Cookies())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	resp.Body.Close()
}

// containsCookie checks if a cookie string contains a specific cookie pair
func containsCookie(cookieHeader, pair string) bool {
	for _, c := range strings.Split(cookieHeader, ";") {
		if strings.TrimSpace(c) == pair {
			return true
		}
	}
	return false
}

func TestCreateMCPToolHandler_BodyTypes(t *testing.T) {
	reqPathParam := []string{"id"}
	reqQueryParam := []string{"q"}
	reqBody := map[string]string{"name": "string", "age": "int", "active": "bool"}
	reqMethod := "post"
	reqHeader := []string{"X-Header"}
	apiCfg := models.ApiConfig{}

	params := map[string]interface{}{
		"id":       "123",
		"q":        "search",
		"name":     "bob",
		"age":      "42",
		"active":   "true",
		"X-Header": "val",
	}
	callReq := mcp.CallToolRequest{Params: struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments,omitempty"`
		Meta      *struct {
			ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
		} `json:"_meta,omitempty"`
	}{Arguments: params}}
	ctx := context.Background()

	// Use httptest to intercept outgoing HTTP requests
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			t.Errorf("Failed to decode body: %v", err)
		}
		if body["name"] != "bob" {
			t.Errorf("Expected name 'bob', got %+v", body["name"])
		}
		ageFloat, ok := body["age"].(float64)
		if !ok || int(ageFloat) != 42 {
			t.Errorf("Expected age 42, got %+v", body["age"])
		}
		activeVal, ok := body["active"].(bool)
		if !ok || !activeVal {
			t.Errorf("Expected active true, got %+v", body["active"])
		}
		if r.Header.Get("X-Header") != "val" {
			t.Errorf("Missing X-Header")
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	reqURL := ts.URL + "/api/{id}"
	h := CreateMCPToolHandler(reqPathParam, reqQueryParam, reqURL, reqBody, reqMethod, reqHeader, apiCfg)
	res, err := h(ctx, callReq)
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}
	if res == nil {
		t.Errorf("Expected non-nil result")
	}
	resultStr := ""
	if res != nil {
		// Marshal the CallToolResult to string and check for "ok"
		b, err := json.Marshal(res)
		if err == nil {
			resultStr = string(b)
		}
	}
	if !strings.Contains(resultStr, "ok") {
		t.Errorf("Expected ok in response, got %s", resultStr)
	}
}
