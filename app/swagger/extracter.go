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
func ExtractSwagger(swaggerSpec models.SwaggerSpec) {
	for path, methods := range swaggerSpec.Paths {
		for method, details := range methods {
			fmt.Printf("Endpoint: %s\n", swaggerSpec.Host+swaggerSpec.BasePath+path)
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

			fmt.Println("Request Body:")
			for _, param := range details.Parameters {
				if param.In == "body" {
					schemaName := ExtractSchemaName(param.Schema.Ref, param.Type)
					fmt.Printf("  - %s (Schema: %s)\n", param.Name, schemaName)
					if definition, found := swaggerSpec.Definitions[schemaName]; found {
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
					schemaName := ExtractSchemaName(resp.Schema.Ref, resp.Schema.Type)
					fmt.Printf("  - Status: %s, Schema: %s\n", status, schemaName)
					if definition, found := swaggerSpec.Definitions[schemaName]; found {
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
