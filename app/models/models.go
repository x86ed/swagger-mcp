package models

import "github.com/mark3labs/mcp-go/server"

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type SwaggerSpec struct {
	// Swagger 2.0 fields
	Host     string `json:"host,omitempty"`
	BasePath string `json:"basePath,omitempty"`
	Swagger  string `json:"swagger,omitempty"`

	// OpenAPI 3.0 fields
	OpenAPI    string      `json:"openapi,omitempty"`
	Servers    []Server    `json:"servers,omitempty"`
	Components *Components `json:"components,omitempty"`

	// Common fields
	Paths       map[string]map[string]Endpoint `json:"paths"`
	Definitions map[string]Definition          `json:"definitions,omitempty"` // Swagger 2.0
}

type Components struct {
	Schemas map[string]Definition `json:"schemas,omitempty"` // OpenAPI 3.0
}

type Definition struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
}

type Property struct {
	Type string `json:"type"`
}

type Endpoint struct {
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	Parameters  []Parameter         `json:"parameters"`
	Responses   map[string]Response `json:"responses"`
	Consumes    []string            `json:"consumes"`
	Produces    []string            `json:"produces"`
}

type Parameter struct {
	Name        string     `json:"name"`
	In          string     `json:"in"`
	Required    bool       `json:"required"`
	Type        string     `json:"type"`
	Schema      *SchemaRef `json:"schema,omitempty"`
	Description string     `json:"description"`
}

type Response struct {
	Description string     `json:"description"`
	Schema      *SchemaRef `json:"schema,omitempty"`
	Type        string     `json:"type,omitempty"`
}

type SchemaRef struct {
	Ref  string `json:"$ref,omitempty"`
	Type string `json:"type,omitempty"`
}

var McpServer *server.MCPServer

var BaseUrl string

var ToolCount int
