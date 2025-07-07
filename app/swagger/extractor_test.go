package swagger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/danishjsheikh/swagger-mcp/app/models"
)

func TestExtractSchemaName(t *testing.T) {
	cases := []struct {
		ref, schemaType, want string
	}{
		{"#/components/schemas/User", "object", "User"},
		{"", "string", "string"},
		{"/foo/bar/Baz", "baz", "Baz"},
	}
	for _, c := range cases {
		got := ExtractSchemaName(c.ref, c.schemaType)
		if got != c.want {
			t.Errorf("ExtractSchemaName(%q, %q) = %q, want %q", c.ref, c.schemaType, got, c.want)
		}
	}
}

func TestGetBaseURL_OpenAPI(t *testing.T) {
	spec := models.SwaggerSpec{
		OpenAPI: "3.0.0",
		Servers: []models.Server{{URL: "https://api.example.com/v1/"}},
	}
	got := getBaseURL(spec)
	want := "https://api.example.com/v1/"
	if got != want {
		t.Errorf("getBaseURL(OpenAPI) = %q, want %q", got, want)
	}
}

func TestGetBaseURL_Swagger2(t *testing.T) {
	spec := models.SwaggerSpec{
		Host:     "api.example.com",
		BasePath: "/v2/",
	}
	got := getBaseURL(spec)
	want := "https://api.example.com/v2/"
	if got != want {
		t.Errorf("getBaseURL(Swagger2) = %q, want %q", got, want)
	}
}

func TestGetBaseURL_Swagger2_NoBasePath(t *testing.T) {
	spec := models.SwaggerSpec{
		Host: "api.example.com",
	}
	got := getBaseURL(spec)
	want := "https://api.example.com"
	if got != want {
		t.Errorf("getBaseURL(Swagger2, no basePath) = %q, want %q", got, want)
	}
}

func TestGetBaseURL_AllBranches(t *testing.T) {
	// OpenAPI 3.0, no servers
	spec := models.SwaggerSpec{OpenAPI: "3.0.0"}
	if got := getBaseURL(spec); got != "" {
		t.Errorf("Expected empty string for OpenAPI 3.0 with no servers, got %q", got)
	}

	// Swagger 2.0, host with http
	spec = models.SwaggerSpec{Host: "http://foo.com"}
	if got := getBaseURL(spec); got != "http://foo.com" {
		t.Errorf("Expected http host to be unchanged, got %q", got)
	}

	// Swagger 2.0, host with https
	spec = models.SwaggerSpec{Host: "https://foo.com"}
	if got := getBaseURL(spec); got != "https://foo.com" {
		t.Errorf("Expected https host to be unchanged, got %q", got)
	}

	// Swagger 2.0, host with no scheme, no basePath
	spec = models.SwaggerSpec{Host: "foo.com"}
	if got := getBaseURL(spec); got != "https://foo.com" {
		t.Errorf("Expected https added to host, got %q", got)
	}

	// Swagger 2.0, host with no scheme, with basePath
	spec = models.SwaggerSpec{Host: "foo.com", BasePath: "/bar/"}
	if got := getBaseURL(spec); got != "https://foo.com/bar/" {
		t.Errorf("Expected https and basePath, got %q", got)
	}
}

func captureOutput(f func()) string {
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	f()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestExtractSwagger_Basic(t *testing.T) {
	spec := models.SwaggerSpec{
		OpenAPI: "3.0.0",
		Servers: []models.Server{{URL: "https://api.example.com/v1/"}},
		Paths: map[string]map[string]models.Endpoint{
			"/users": {
				"get": models.Endpoint{
					Summary:     "List users",
					Description: "Returns a list of users.",
					Parameters: []models.Parameter{
						{Name: "X-Auth", In: "header", Required: true},
						{Name: "id", In: "path", Required: true, Type: "string", Description: "User ID"},
					},
					Responses: map[string]models.Response{
						"200": {Description: "OK", Type: "array"},
					},
				},
			},
		},
	}

	output := captureOutput(func() { ExtractSwagger(spec) })

	if !strings.Contains(output, "Endpoint: https://api.example.com/v1/users") {
		t.Errorf("Expected endpoint in output, got: %s", output)
	}
	if !strings.Contains(output, "Method: GET") {
		t.Errorf("Expected method in output, got: %s", output)
	}
	if !strings.Contains(output, "Summary: List users") {
		t.Errorf("Expected summary in output, got: %s", output)
	}
	if !strings.Contains(output, "Description: Returns a list of users.") {
		t.Errorf("Expected description in output, got: %s", output)
	}
	if !strings.Contains(output, "- X-Auth (Required: true)") {
		t.Errorf("Expected header param in output, got: %s", output)
	}
	if !strings.Contains(output, "- id (Required: true, Type: string)") {
		t.Errorf("Expected path param in output, got: %s", output)
	}
	if !strings.Contains(output, "Description: User ID") {
		t.Errorf("Expected path param description in output, got: %s", output)
	}
	if !strings.Contains(output, "Status 200:") {
		t.Errorf("Expected response status in output, got: %s", output)
	}
	if !strings.Contains(output, "Type: array") {
		t.Errorf("Expected response type in output, got: %s", output)
	}
}

func TestExtractSwagger_RequestAndResponseBody(t *testing.T) {
	spec := models.SwaggerSpec{
		Swagger:  "2.0",
		Host:     "api.example.com",
		BasePath: "/v2/",
		Paths: map[string]map[string]models.Endpoint{
			"/widgets": {
				"post": models.Endpoint{
					Summary:     "Create widget",
					Description: "Creates a new widget.",
					Parameters: []models.Parameter{
						{Name: "body", In: "body", Required: true, Type: "object", Schema: &models.SchemaRef{Ref: "#/definitions/Widget"}},
					},
					Responses: map[string]models.Response{
						"201": {Description: "Created", Schema: &models.SchemaRef{Ref: "#/definitions/Widget"}},
					},
				},
			},
		},
		Definitions: map[string]models.Definition{
			"Widget": {
				Type: "object",
				Properties: map[string]models.Property{
					"name": {Type: "string"},
					"size": {Type: "int"},
				},
			},
		},
	}

	output := captureOutput(func() { ExtractSwagger(spec) })

	if !strings.Contains(output, "Endpoint: https://api.example.com/v2/widgets") {
		t.Errorf("Expected endpoint in output, got: %s", output)
	}
	if !strings.Contains(output, "Method: POST") {
		t.Errorf("Expected method in output, got: %s", output)
	}
	if !strings.Contains(output, "Request Body:") || !strings.Contains(output, "Schema: Widget") {
		t.Errorf("Expected request body schema in output, got: %s", output)
	}
	if !strings.Contains(output, "- name: string") || !strings.Contains(output, "- size: int") {
		t.Errorf("Expected request body properties in output, got: %s", output)
	}
	if !strings.Contains(output, "Response Body:") || !strings.Contains(output, "Schema: Widget") {
		t.Errorf("Expected response body schema in output, got: %s", output)
	}
	if !strings.Contains(output, "- name: string") || !strings.Contains(output, "- size: int") {
		t.Errorf("Expected response body properties in output, got: %s", output)
	}
}

func TestExtractSwagger_AllBranches(t *testing.T) {
	spec := models.SwaggerSpec{
		Swagger:  "2.0",
		Host:     "api.example.com",
		BasePath: "/v2/",
		Paths: map[string]map[string]models.Endpoint{
			"/empty": {
				"get": models.Endpoint{
					Summary:     "No params",
					Description: "No parameters or responses.",
					Parameters:  []models.Parameter{},
					Responses:   map[string]models.Response{},
				},
			},
			"/body": {
				"post": models.Endpoint{
					Summary:     "Body param",
					Description: "Body param with missing schema def.",
					Parameters: []models.Parameter{
						{Name: "body", In: "body", Required: true, Type: "object", Schema: &models.SchemaRef{Ref: "#/definitions/NotFound"}},
					},
					Responses: map[string]models.Response{
						"400": {Description: "Bad req", Schema: &models.SchemaRef{Ref: "#/definitions/NotFound"}},
					},
				},
			},
			"/respType": {
				"get": models.Endpoint{
					Summary:     "Resp type",
					Description: "Response with type only.",
					Parameters:  []models.Parameter{},
					Responses: map[string]models.Response{
						"200": {Type: "string"},
					},
				},
			},
			"/respNoSchema": {
				"get": models.Endpoint{
					Summary:     "No schema",
					Description: "Response with no schema/type.",
					Parameters:  []models.Parameter{},
					Responses: map[string]models.Response{
						"204": {},
					},
				},
			},
		},
	}

	output := captureOutput(func() { ExtractSwagger(spec) })

	if !strings.Contains(output, "Endpoint: https://api.example.com/v2/empty") {
		t.Errorf("Expected /empty endpoint in output")
	}
	if !strings.Contains(output, "No parameters or responses.") {
		t.Errorf("Expected description for /empty")
	}
	if !strings.Contains(output, "Request Body:") {
		t.Errorf("Expected request body section for /body")
	}
	if !strings.Contains(output, "Schema: NotFound") {
		t.Errorf("Expected missing schema name for /body")
	}
	if !strings.Contains(output, "Type: string") {
		t.Errorf("Expected response type for /respType")
	}
	if !strings.Contains(output, "No response schema defined") {
		t.Errorf("Expected no response schema for /respNoSchema")
	}
}

func TestExtractSchemaName_AllBranches(t *testing.T) {
	if got := ExtractSchemaName("", "foo"); got != "foo" {
		t.Errorf("Expected fallback to schemaType, got %q", got)
	}
	if got := ExtractSchemaName("/a/b/c", "foo"); got != "c" {
		t.Errorf("Expected last part of ref, got %q", got)
	}
	if got := ExtractSchemaName("/", "foo"); got != "" {
		t.Errorf("Expected empty string for ref '/', got %q", got)
	}
}

func TestExtractSwagger_ResponseSchemaTypeBranch(t *testing.T) {
	spec := models.SwaggerSpec{
		Swagger:  "2.0",
		Host:     "api.example.com",
		BasePath: "/v2/",
		Paths: map[string]map[string]models.Endpoint{
			"/respSchemaType": {
				"get": models.Endpoint{
					Summary:     "Resp schema type only",
					Description: "Response with schema type only.",
					Parameters:  []models.Parameter{},
					Responses: map[string]models.Response{
						"200": {Description: "OK", Schema: &models.SchemaRef{Type: "string"}},
					},
				},
			},
		},
		Definitions: map[string]models.Definition{}, // No "string" definition
	}

	output := captureOutput(func() { ExtractSwagger(spec) })

	if !strings.Contains(output, "Type: string") {
		t.Errorf("Expected 'Type: string' in output, got: %s", output)
	}
}
