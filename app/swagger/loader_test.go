package swagger

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/danishjsheikh/swagger-mcp/app/models"
)

func TestLoadSwagger_File_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "swagger-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	spec := models.SwaggerSpec{Swagger: "2.0", Host: "example.com"}
	data, _ := json.Marshal(spec)
	if _, err := tmpFile.Write(data); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	got, err := LoadSwagger("file://" + tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadSwagger file success: %v", err)
	}
	if got.Host != "example.com" {
		t.Errorf("expected host 'example.com', got %q", got.Host)
	}
}

func TestLoadSwagger_File_ReadError(t *testing.T) {
	_, err := LoadSwagger("file:///nonexistent/path/to/spec.json")
	if err == nil || !strings.Contains(err.Error(), "error reading file") {
		t.Errorf("expected file read error, got %v", err)
	}
}

func TestLoadSwagger_HTTP_Success(t *testing.T) {
	spec := models.SwaggerSpec{Swagger: "2.0", Host: "httphost.com"}
	data, _ := json.Marshal(spec)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	got, err := LoadSwagger(ts.URL)
	if err != nil {
		t.Fatalf("LoadSwagger http success: %v", err)
	}
	if got.Host != "httphost.com" {
		t.Errorf("expected host 'httphost.com', got %q", got.Host)
	}
}

func TestLoadSwagger_HTTP_GetError(t *testing.T) {
	// Use an invalid URL to force http.Get error
	_, err := LoadSwagger("http://invalid.invalid")
	if err == nil || !strings.Contains(err.Error(), "error getting spec") {
		t.Errorf("expected http get error, got %v", err)
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestLoadSwagger_HTTP_ReadError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		// hijack the connection to simulate a broken body
	}))
	defer ts.Close()

	// Create a custom client to inject a broken body
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(errorReader{}),
	}

	client := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return resp, nil
		}),
	}

	oldDefaultClient := http.DefaultClient
	http.DefaultClient = client
	defer func() { http.DefaultClient = oldDefaultClient }()

	_, err := LoadSwagger(ts.URL)
	if err == nil || !strings.Contains(err.Error(), "error reading spec") {
		t.Errorf("expected error reading spec, got %v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestLoadSwagger_JSONError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	_, err := LoadSwagger(ts.URL)
	if err == nil || !strings.Contains(err.Error(), "error parsing JSON") {
		t.Errorf("expected json parse error, got %v", err)
	}
}

func TestLoadSwagger_HTTP_StatusError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer ts.Close()

	_, err := LoadSwagger(ts.URL)
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Errorf("expected status error, got %v", err)
	}
}

func TestLoadSwagger_PlainFilePath_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "swagger-plain-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	spec := models.SwaggerSpec{Swagger: "2.0", Host: "plainfile.com"}
	data, _ := json.Marshal(spec)
	if _, err := tmpFile.Write(data); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	got, err := LoadSwagger(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadSwagger plain file path: %v", err)
	}
	if got.Host != "plainfile.com" {
		t.Errorf("expected host 'plainfile.com', got %q", got.Host)
	}
}

func TestLoadSwagger_SizeLimit(t *testing.T) {
	SetMaxSpecSize(100) // 100 bytes
	tmpFile, err := os.CreateTemp("", "swagger-big-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	big := make([]byte, 200) // 200 bytes
	for i := range big {
		big[i] = 'a'
	}
	if _, err := tmpFile.Write(big); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	_, err = LoadSwagger(tmpFile.Name())
	if err == nil || !strings.Contains(err.Error(), "spec file too large") {
		t.Errorf("expected size limit error, got %v", err)
	}

	SetMaxSpecSize(DefaultMaxSpecSize) // reset
}

func TestLoadSwagger_SizeLimitEnvVar(t *testing.T) {
	os.Setenv("SWAGGER_MCP_MAX_SPEC_SIZE", "150")
	defer os.Unsetenv("SWAGGER_MCP_MAX_SPEC_SIZE")
	SetMaxSpecSize(-1) // reset to allow env var to take effect
	max := GetMaxSpecSize()
	if max != 150 {
		t.Errorf("expected max spec size 150 from env, got %d", max)
	}
}
