package swaggerloader

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/danishjsheikh/swagger-mcp/app/models"
)

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
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	var swagger models.SwaggerSpec
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
