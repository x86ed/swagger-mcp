package models

import "github.com/mark3labs/mcp-go/server"

type SwaggerSpec struct {
	Host        string                         `json:"host"`
	BasePath    string                         `json:"basePath"`
	Paths       map[string]map[string]Endpoint `json:"paths"`
	Definitions map[string]Definition          `json:"definitions"`
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

type Definition struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
}

type Property struct {
	Type string `json:"type"`
}

var McpServer *server.MCPServer

var BaseUrl string
