package swagger

import (
	"fmt"
	"strings"

	"github.com/danishjsheikh/swagger-mcp/app/models"
)

func ExtractSchemaName(ref, schemaType string) string {
	if ref != "" {
		parts := strings.Split(ref, "/")
		return parts[len(parts)-1]
	}
	return schemaType
}

func getBaseURL(swaggerSpec models.SwaggerSpec) string {
	// For OpenAPI 3.0
	if swaggerSpec.OpenAPI != "" && len(swaggerSpec.Servers) > 0 {
		return strings.TrimSuffix(swaggerSpec.Servers[0].URL, "/")
	}
	
	// For Swagger 2.0
	baseURL := swaggerSpec.Host
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	if swaggerSpec.BasePath != "" {
		baseURL = strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(swaggerSpec.BasePath, "/")
	}
	return baseURL
}

func ExtractSwagger(swaggerSpec models.SwaggerSpec) {
	baseURL := getBaseURL(swaggerSpec)
	
	for path, methods := range swaggerSpec.Paths {
		for method, details := range methods {
			fullURL := strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(path, "/")
			fmt.Printf("\nEndpoint: %s\n", fullURL)
			fmt.Printf("Method: %s\n", strings.ToUpper(method))
			fmt.Printf("Summary: %s\n", details.Summary)
			fmt.Printf("Description: %s\n", details.Description)
			
			fmt.Println("\nHeaders:")
			for _, param := range details.Parameters {
				if param.In == "header" {
					fmt.Printf("  - %s (Required: %t)\n", param.Name, param.Required)
				}
			}
			
			fmt.Println("\nPath Parameters:")
			for _, param := range details.Parameters {
				if param.In == "path" {
					fmt.Printf("  - %s (Required: %t, Type: %s)\n", param.Name, param.Required, param.Type)
					if param.Description != "" {
						fmt.Printf("    Description: %s\n", param.Description)
					}
				}
			}

			fmt.Println("\nRequest Body:")
			for _, param := range details.Parameters {
				if param.In == "body" {
					schemaName := ExtractSchemaName(param.Schema.Ref, param.Type)
					fmt.Printf("  Schema: %s\n", schemaName)
					if definition, found := swaggerSpec.Definitions[schemaName]; found {
						for propName, prop := range definition.Properties {
							fmt.Printf("    - %s: %s\n", propName, prop.Type)
						}
					} else if schemaName != "" {
						fmt.Printf("    Type: %s\n", schemaName)
					}
				}
			}
			
			fmt.Println("\nResponse Body:")
			for status, resp := range details.Responses {
				fmt.Printf("  Status %s:\n", status)
				if resp.Schema != nil {
					schemaName := ExtractSchemaName(resp.Schema.Ref, resp.Schema.Type)
					if definition, found := swaggerSpec.Definitions[schemaName]; found {
						fmt.Printf("    Schema: %s\n", schemaName)
						for propName, prop := range definition.Properties {
							fmt.Printf("      - %s: %s\n", propName, prop.Type)
						}
					} else if resp.Schema.Type != "" {
						fmt.Printf("    Type: %s\n", resp.Schema.Type)
					} else {
						fmt.Printf("    Schema Reference: %s\n", resp.Schema.Ref)
					}
				} else if resp.Type != "" {
					fmt.Printf("    Type: %s\n", resp.Type)
				} else {
					fmt.Printf("    No response schema defined\n")
				}
				if resp.Description != "" {
					fmt.Printf("    Description: %s\n", resp.Description)
				}
			}
			fmt.Println("\n----------------------------")
		}
	}
}
