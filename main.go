package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

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

func extractSchemaName(ref, schemaType string) string {
	if ref != "" {
		parts := strings.Split(ref, "/")
		return parts[len(parts)-1]
	}
	return schemaType
}

func GetSwaggerDef() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <swagger_json_file>")
		return
	}

	filePath := os.Args[1]
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	var swagger SwaggerSpec
	if err := json.Unmarshal(data, &swagger); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	for path, methods := range swagger.Paths {
		for method, details := range methods {
			fmt.Printf("Endpoint: %s\n", swagger.Host+swagger.BasePath+path)
			fmt.Printf("Method: %s\n", method)
			fmt.Printf("Summary: %s\n", details.Summary)
			fmt.Printf("Description: %s\n", details.Description)
			fmt.Println("Headers:")
			for _, param := range details.Parameters {
				if param.In == "header" {
					fmt.Printf("  - %s (Required: %t)\n", param.Name, param.Required)
				}
			}
			fmt.Println("Path Paramters:")
			for _, param := range details.Parameters {
				if param.In == "path" {
					fmt.Printf("  - %s (Type: %s, Required: %t)\n", param.Name, param.Type, param.Required)
				}
			}
			fmt.Println("Request Body:")
			for _, param := range details.Parameters {
				if param.In == "body" {
					schemaName := extractSchemaName(param.Schema.Ref, param.Type)
					fmt.Printf("  - %s (Schema: %s)\n", param.Name, schemaName)
					if definition, found := swagger.Definitions[schemaName]; found {
						for propName, prop := range definition.Properties {
							fmt.Printf("    - %s: %s\n", propName, prop.Type)
						}
					} else if schemaName != "" {
						fmt.Printf("    - Type: %s\n", schemaName)
					}
				}
			}
			fmt.Println("Response Body:")
			for status, resp := range details.Responses {
				if resp.Schema != nil {
					schemaName := extractSchemaName(resp.Schema.Ref, resp.Schema.Type)
					fmt.Printf("  - Status: %s, Schema: %s\n", status, schemaName)
					if definition, found := swagger.Definitions[schemaName]; found {
						for propName, prop := range definition.Properties {
							fmt.Printf("    - %s: %s\n", propName, prop.Type)
						}
					} else if schemaName != "" {
						fmt.Printf("    - Type: %s\n", schemaName)
					}
				} else {
					fmt.Printf("  - Status: %s, No schema defined\n", status)
				}
			}
			fmt.Println("----------------------------")
		}
	}
}

func main() {
	GetSwaggerDef()

}
