package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/danishjsheikh/swagger-mcp/app/models"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func ExtractSchemaName(ref, schemaType string) string {
	if ref != "" {
		parts := strings.Split(ref, "/")
		return parts[len(parts)-1]
	}
	return schemaType
}

func CreateServer(swaggerSpec models.SwaggerSpec) {
	sseMode := flag.Bool("sse", false, "Run in SSE mode instead of stdio mode")
	flag.Parse()

	models.McpServer = server.NewMCPServer(
		"swagegr-mcp",
		"1.0.0",
	)

	LoadSwaggerServer(swaggerSpec)

	if *sseMode {
		// Create and start SSE server
		sseServer := server.NewSSEServer(models.McpServer, "http://localhost:8080")
		log.Printf("Starting SSE server on localhost:8080")
		if err := sseServer.Start(":8080"); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		// Run as stdio server
		if err := server.ServeStdio(models.McpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

func LoadSwaggerServer(swaggerSpec models.SwaggerSpec) {
	for path, methods := range swaggerSpec.Paths {

		for method, details := range methods {
			var expectedResponse []string
			var toolOption []mcp.ToolOption
			var reqURL string
			var reqMethod string
			reqBody := make(map[string]string)
			var reqPathParam []string
			var reqHeader []string
			reqMethod = fmt.Sprint(method)
			reqURL = fmt.Sprint(swaggerSpec.Host + swaggerSpec.BasePath + path)

			for _, param := range details.Parameters {
				if param.In == "header" {
					if param.Required {
						toolOption = append(toolOption, mcp.WithString(
							fmt.Sprint(param.Name),
							mcp.Description(fmt.Sprintf("The data for %s", param.Name)),
							mcp.Required(),
						))
					} else {
						toolOption = append(toolOption, mcp.WithString(
							fmt.Sprint(param.Name),
							mcp.Description(fmt.Sprintf("The data for %s", param.Name)),
						))
					}
					reqHeader = append(reqHeader, param.Name)
				}
			}

			for _, param := range details.Parameters {
				if param.In == "path" {
					if param.Required {
						toolOption = append(toolOption, mcp.WithString(
							fmt.Sprint(param.Name),
							mcp.Description(fmt.Sprintf("The data for %s", param.Name)),
							mcp.Required(),
						))
					} else {
						toolOption = append(toolOption, mcp.WithString(
							fmt.Sprint(param.Name),
							mcp.Description(fmt.Sprintf("The data for %s", param.Name)),
						))
					}
					reqPathParam = append(reqPathParam, param.Name)
				}
			}
			for _, param := range details.Parameters {
				if param.In == "body" {
					schemaName := ExtractSchemaName(param.Schema.Ref, param.Type)
					if definition, found := swaggerSpec.Definitions[schemaName]; found {
						for propName, prop := range definition.Properties {
							toolOption = append(toolOption, mcp.WithString(
								fmt.Sprintf("%s", propName),
								mcp.Description(fmt.Sprintf("The data for %s, it should be in format of %s", propName, prop.Type)),
								mcp.Required(),
							))
							reqBody[propName] = prop.Type
						}
					}
				}
			}
			for status, resp := range details.Responses {
				if resp.Schema != nil {
					schemaName := ExtractSchemaName(resp.Schema.Ref, resp.Schema.Type)
					if definition, found := swaggerSpec.Definitions[schemaName]; found {
						defData, _ := json.Marshal(definition)
						expectedResponse = append(expectedResponse, fmt.Sprintf(`{status_code: %s, response_body:%s}`, status, string(defData)))
					}
				}
			}

			toolOption = append(toolOption, mcp.WithDescription(fmt.Sprintf(`Only use this tool when you dont have any information about  %s or %s, or need you perfrom the opertation of %s or %s, Use this tool only when no other tool can perfrom the oprtation of %s or %s,  You will get response as one of %s, the response is only for refernce`,
				details.Summary, details.Description, details.Summary, details.Description, details.Summary, details.Description, strings.Join(expectedResponse, ", "))))

			models.McpServer.AddTool(mcp.NewTool(
				fmt.Sprintf("%s_%s", method, path),
				toolOption...,
			), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				for _, paramName := range reqPathParam {
					param, ok := request.Params.Arguments[paramName].(string)
					if !ok {
						return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid Path Parameter: %s", paramName)), nil
					}
					reqURL = strings.Replace(reqURL, fmt.Sprintf("{%s}", paramName), param, 1)
				}

				reqBodyData := make(map[string]interface{})
				for paramName, paramType := range reqBody {
					param, exists := request.Params.Arguments[paramName]
					if !exists {
						return mcp.NewToolResultError(fmt.Sprintf("Missing Body Parameter: %s", paramName)), nil
					}

					switch paramType {
					case "string":
						if value, ok := param.(string); ok {
							reqBodyData[paramName] = value
						} else {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected string", paramName)), nil
						}
					case "int":
						switch value := param.(type) {
						case int:
							reqBodyData[paramName] = value
						case float64:
							reqBodyData[paramName] = int(value)
						default:
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected int", paramName)), nil
						}
					case "float":
						if value, ok := param.(float64); ok {
							reqBodyData[paramName] = value
						} else {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected float", paramName)), nil
						}
					case "bool":
						if value, ok := param.(bool); ok {
							reqBodyData[paramName] = value
						} else {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected bool", paramName)), nil
						}
					case "array":
						if value, ok := param.([]interface{}); ok {
							reqBodyData[paramName] = value
						} else {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected array", paramName)), nil
						}
					case "object":
						if value, ok := param.(map[string]interface{}); ok {
							reqBodyData[paramName] = value
						} else {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected object", paramName)), nil
						}
					default:
						return mcp.NewToolResultError(fmt.Sprintf("Unsupported parameter type: %s for %s", paramType, paramName)), nil
					}
				}

				reqBodyDataBytes, err := json.Marshal(reqBodyData)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal request body: %v", err)), nil
				}

				req, err := http.NewRequest(reqMethod, reqURL, bytes.NewBuffer(reqBodyDataBytes))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to create HTTP request: %v", err)), nil
				}

				for _, headerName := range reqHeader {
					headerValue, ok := request.Params.Arguments[headerName].(string)
					if !ok {
						return mcp.NewToolResultError(fmt.Sprintf("Missing or invalid Header: %s", headerName)), nil
					}
					req.Header.Add(headerName, headerValue)
				}

				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to make HTTP request: %v", err)), nil
				}

				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to read HTTP Response: %v", err)), nil
				}
				fmt.Println(string(body))
				return mcp.NewToolResultText(string(body)), nil
			})
		}

	}
}
