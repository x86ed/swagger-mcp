package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
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

func CreateServer(swaggerSpec models.SwaggerSpec, sseMode bool, baseUrl string, port int) {
	models.McpServer = server.NewMCPServer(
		"swagegr-mcp",
		"1.0.0",
	)

	LoadSwaggerServer(swaggerSpec)

	if sseMode {
		// Create and start SSE server
		sseServer := server.NewSSEServer(models.McpServer, baseUrl)
		log.Printf("Starting SSE server on %s:%d", baseUrl, port)
		if err := sseServer.Start(fmt.Sprintf(":%d", port)); err != nil {
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

			toolOption = append(toolOption, mcp.WithDescription(fmt.Sprintf(`Use this tool only when the request exactly matches %s or %s. If you dont have any of the required parameters then always ask user for it, *Dont fill any paramter on your own or keep it empty*. If there is [Error], only state that error in your reponse and stop the reponse there itself. *Do not ever maintain records in your memory for eg list of users or orders*`,
				details.Summary, details.Description)))

			models.McpServer.AddTool(mcp.NewTool(
				fmt.Sprintf("%s_%s", method, strings.ReplaceAll(strings.ReplaceAll(path, "}", ""), "{", "")),
				toolOption...,
			), CreateMCPToolHandler(reqPathParam, reqURL, reqBody, reqMethod, reqHeader))
		}
	}
}

func CreateMCPToolHandler(reqPathParam []string, reqURL string, reqBody map[string]string, reqMethod string, reqHeader []string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		currentReqURL := reqURL
		for _, paramName := range reqPathParam {
			param, ok := request.Params.Arguments[paramName].(string)
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("[Error] missing or invalid Path Parameter: %s", paramName)), nil
			}
			currentReqURL = strings.Replace(currentReqURL, fmt.Sprintf("{%s}", paramName), param, 1)
		}

		reqBodyData := make(map[string]interface{})
		for paramName, paramType := range reqBody {
			paramStr, exists := request.Params.Arguments[paramName].(string)
			if !exists {
				return mcp.NewToolResultError(fmt.Sprintf("[Error] missing Body Parameter: %s", paramName)), nil
			}

			switch paramType {
			case "string":
				reqBodyData[paramName] = paramStr

			case "int", "integer":
				intValue, err := strconv.Atoi(paramStr)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("[Error] invalid type for parameter %s, expected int", paramName)), nil
				}
				reqBodyData[paramName] = intValue

			case "float":
				floatValue, err := strconv.ParseFloat(paramStr, 64)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("[Error] invalid type for parameter %s, expected float", paramName)), nil
				}
				reqBodyData[paramName] = floatValue

			case "bool", "boolean":
				boolValue, err := strconv.ParseBool(paramStr)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("[Error] invalid type for parameter %s, expected bool", paramName)), nil
				}
				reqBodyData[paramName] = boolValue

			case "array":
				var arrayValue []interface{}
				if err := json.Unmarshal([]byte(paramStr), &arrayValue); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("[Error] invalid type for parameter %s, expected array", paramName)), nil
				}
				reqBodyData[paramName] = arrayValue

			case "object":
				var objectValue map[string]interface{}
				if err := json.Unmarshal([]byte(paramStr), &objectValue); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("[Error] invalid type for parameter %s, expected object", paramName)), nil
				}
				reqBodyData[paramName] = objectValue

			default:
				return mcp.NewToolResultError(fmt.Sprintf("[Error] unsupported parameter type: %s for %s", paramType, paramName)), nil
			}

		}
		reqBodyDataBytes, err := json.Marshal(reqBodyData)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("[Error] failed to marshal request body: %v", err)), nil
		}

		req, err := http.NewRequest(strings.ToUpper(reqMethod), currentReqURL, bytes.NewBuffer(reqBodyDataBytes))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("[Error] failed to create HTTP request: %v", err)), nil
		}

		for _, headerName := range reqHeader {
			headerValue, ok := request.Params.Arguments[headerName].(string)
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("[Error] missing or invalid Header: %s", headerName)), nil
			}
			req.Header.Add(headerName, headerValue)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("[Error] failed to make HTTP request: %v", err)), nil
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("[Error] failed to read HTTP Response: %v", err)), nil
		}
		fmt.Println(string(body))
		return mcp.NewToolResultText(string(body)), nil
	}
}
