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
	"strconv"
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
			expectedResponse := []string{}
			toolOption := []mcp.ToolOption{}
			reqURL := fmt.Sprintf("http://%s%s%s", swaggerSpec.Host, swaggerSpec.BasePath, path)
			reqMethod := fmt.Sprint(method)
			reqBody := make(map[string]string)
			reqPathParam := []string{}
			reqHeader := []string{}

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
								fmt.Sprint(propName),
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
				} else if resp.Type != "" {
					expectedResponse = append(expectedResponse, fmt.Sprintf(`{status_code: %s, response_body:%s}`, status, string(resp.Type)))
				}
			}

			toolOption = append(toolOption, mcp.WithDescription(fmt.Sprintf(`Use this tool *only* when need to performe the operation of %s or %s. Do *not* use this tool if any other tool can perform the operation of %s or %s, Dont rely on stored history as data can be changed externally too,

            Your response must strictly be one of: %s. If an error occurs, state the error clearly without modifying or guessing the response. Do *not* hallucinate, generate random data, or make assumptions beyond the error message or response received.
            
            Always respond based *only* on the received error or expected response. You are a precise and reliable assistant.`,
				details.Summary, details.Description,
				details.Summary, details.Description,
				strings.Join(expectedResponse, ", "))))

			models.McpServer.AddTool(mcp.NewTool(
				fmt.Sprintf("%s_%s", method, strings.ReplaceAll(strings.ReplaceAll(path, "}", ""), "{", "")),
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
					paramStr, exists := request.Params.Arguments[paramName].(string)
					if !exists {
						return mcp.NewToolResultError(fmt.Sprintf("Missing Body Parameter: %s", paramName)), nil
					}

					switch paramType {
					case "string":
						reqBodyData[paramName] = paramStr

					case "int", "integer":
						intValue, err := strconv.Atoi(paramStr)
						if err != nil {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected int", paramName)), nil
						}
						reqBodyData[paramName] = intValue

					case "float":
						floatValue, err := strconv.ParseFloat(paramStr, 64)
						if err != nil {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected float", paramName)), nil
						}
						reqBodyData[paramName] = floatValue

					case "bool", "boolean":
						boolValue, err := strconv.ParseBool(paramStr)
						if err != nil {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected bool", paramName)), nil
						}
						reqBodyData[paramName] = boolValue

					case "array":
						var arrayValue []interface{}
						if err := json.Unmarshal([]byte(paramStr), &arrayValue); err != nil {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected array", paramName)), nil
						}
						reqBodyData[paramName] = arrayValue

					case "object":
						var objectValue map[string]interface{}
						if err := json.Unmarshal([]byte(paramStr), &objectValue); err != nil {
							return mcp.NewToolResultError(fmt.Sprintf("Invalid type for parameter %s, expected object", paramName)), nil
						}
						reqBodyData[paramName] = objectValue

					default:
						return mcp.NewToolResultError(fmt.Sprintf("Unsupported parameter type: %s for %s", paramType, paramName)), nil
					}

				}
				reqBodyDataBytes, err := json.Marshal(reqBodyData)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal request body: %v", err)), nil
				}

				req, err := http.NewRequest(strings.ToUpper(reqMethod), reqURL, bytes.NewBuffer(reqBodyDataBytes))
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
