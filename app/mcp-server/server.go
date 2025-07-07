// Package mcpserver provides functions to dynamically generate MCP tools and HTTP handlers
// from OpenAPI/Swagger specifications, supporting both SSE and stdio server modes.
// It includes utilities for path/method filtering, security handling, and request/response mapping.
package mcpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/danishjsheikh/swagger-mcp/app/models"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// sseHeadersKey is the context key for passing SSE headers.
const sseHeadersKey = "__sseHeadersKey"

// ExtractSchemaName returns the schema name from a JSON reference string or falls back to the schema type.
func ExtractSchemaName(ref, schemaType string) string {
	if ref != "" {
		parts := strings.Split(ref, "/")
		return parts[len(parts)-1]
	}
	return schemaType
}

// compileRegexes splits a comma-separated string of regex patterns and compiles them.
// Invalid patterns are skipped with a log message.
func compileRegexes(paths string) []*regexp.Regexp {
	var regexes []*regexp.Regexp
	for _, path := range strings.Split(paths, ",") {
		if path = strings.TrimSpace(path); path != "" {
			regex, err := regexp.Compile(path)
			if err != nil {
				log.Printf("Invalid regex pattern: %s, error: %v", path, err)
				continue
			}
			regexes = append(regexes, regex)
		}
	}
	return regexes
}

// shouldIncludePath determines if a path should be included based on include/exclude regexes.
// If no include regexes are provided, all paths are included by default.
func shouldIncludePath(path string, includeRegexes, excludeRegexes []*regexp.Regexp) bool {
	include := len(includeRegexes) == 0
	for _, regex := range includeRegexes {
		if regex.MatchString(path) {
			include = true
			break
		}
	}
	if !include {
		return false
	}
	for _, regex := range excludeRegexes {
		if regex.MatchString(path) {
			return false
		}
	}
	return true
}

// shouldIncludeMethod determines if an HTTP method should be included based on include/exclude lists.
// If no include methods are provided, all methods are included by default.
func shouldIncludeMethod(method string, includeMethods, excludeMethods []string) bool {
	include := len(includeMethods) == 0
	for _, m := range includeMethods {
		if strings.EqualFold(strings.TrimSpace(m), method) {
			include = true
			break
		}
	}
	if !include {
		return false
	}
	for _, m := range excludeMethods {
		if strings.EqualFold(strings.TrimSpace(m), method) {
			return false
		}
	}
	return true
}

// CreateServer creates and starts an MCP server from a Swagger/OpenAPI spec and config.
// It supports both SSE and stdio server modes.
func CreateServer(swaggerSpec models.SwaggerSpec, config models.Config) {
	apiVersion := "1.0.0"
	if swaggerSpec.Info != nil && swaggerSpec.Info.Version != "" {
		apiVersion = swaggerSpec.Info.Version
	}
	mcpServer := server.NewMCPServer(
		"swagger-mcp",
		apiVersion,
	)
	LoadSwaggerServer(mcpServer, swaggerSpec, config.ApiCfg)
	if config.SseCfg.SseMode {
		// Create and start SSE server
		sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(config.SseCfg.SseUrl), server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			if len(config.ApiCfg.SseHeaders) == 0 {
				return ctx
			}
			keys := strings.Split(config.ApiCfg.SseHeaders, ",")
			sseHeaders := map[string]string{}
			for _, key := range keys {
				sseHeaders[key] = r.Header.Get(key)
			}
			return context.WithValue(ctx, sseHeadersKey, sseHeaders)
		}))
		endpoint, err := sseServer.CompleteSseEndpoint()
		if err != nil {
			log.Fatalf("Error creating SSE endpoint: %v", err)
		}
		log.Printf("Starting SSE server on %s, endpoint: %s", config.SseCfg.SseAddr, endpoint)
		if err := sseServer.Start(config.SseCfg.SseAddr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		// Run as stdio server
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// LoadSwaggerServer registers tools and handlers on the MCP server for each path/method in the Swagger spec.
// It applies path/method filtering and builds tool options and handlers for each endpoint.
func LoadSwaggerServer(mcpServer *server.MCPServer, swaggerSpec models.SwaggerSpec, apiCfg models.ApiConfig) {
	includeRegexes := compileRegexes(apiCfg.IncludePaths)
	excludeRegexes := compileRegexes(apiCfg.ExcludePaths)
	includedMethods := []string{}
	if len(strings.TrimSpace(apiCfg.IncludeMethods)) > 0 {
		includedMethods = strings.Split(apiCfg.IncludeMethods, ",")
	}
	excludedMethods := []string{}
	if len(strings.TrimSpace(apiCfg.ExcludeMethods)) > 0 {
		excludedMethods = strings.Split(apiCfg.ExcludeMethods, ",")
	}

	for path, methods := range swaggerSpec.Paths {

		if !shouldIncludePath(path, includeRegexes, excludeRegexes) {
			continue
		}

		for method, details := range methods {
			if !shouldIncludeMethod(method, includedMethods, excludedMethods) {
				continue
			}
			expectedResponse := []string{}
			toolOption := []mcp.ToolOption{}

			var reqURL string
			var baseURL string

			if apiCfg.BaseUrl == "" {
				// Determine base URL based on version
				if swaggerSpec.OpenAPI != "" {
					// OpenAPI 3.0
					if len(swaggerSpec.Servers) > 0 {
						baseURL = strings.TrimSuffix(swaggerSpec.Servers[0].URL, "/")
					} else {
						baseURL = "/" // Default to relative path if no servers defined
					}
				} else {
					// Swagger 2.0
					baseURL = swaggerSpec.Host
					if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
						baseURL = "https://" + baseURL
					}
					if swaggerSpec.BasePath != "" {
						baseURL = strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(swaggerSpec.BasePath, "/")
					}
				}
			} else {
				baseURL = apiCfg.BaseUrl
			}

			reqURL = strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(path, "/")

			reqMethod := fmt.Sprint(method)
			reqBody := make(map[string]string)
			reqPathParam := []string{}
			reqQueryParam := []string{}
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
				if param.In == "query" {
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
					reqQueryParam = append(reqQueryParam, param.Name)
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

			toolName := fmt.Sprintf("%s_%s", method, strings.ReplaceAll(strings.ReplaceAll(path, "}", ""), "{", ""))

			mcpServer.AddTool(
				mcp.NewTool(toolName, toolOption...),
				CreateMCPToolHandler(
					reqPathParam, reqQueryParam, reqURL, reqBody, reqMethod, reqHeader, apiCfg,
				),
			)
		}
	}
}

// setRequestSecurity sets authentication headers, query params, or cookies on the request
// based on the security type and provided credentials.
func setRequestSecurity(req *http.Request, security string, basicAuth string, apiKeyAuth string, bearerAuth string) {
	securityType := strings.TrimSpace(security)

	// Fix: always use correct argument for each security type, regardless of what is passed in others
	switch securityType {
	case "basic":
		if basicAuth != "" {
			auth := base64.StdEncoding.EncodeToString([]byte(basicAuth))
			req.Header.Set("Authorization", "Basic "+auth)
		}
	case "bearer":
		if bearerAuth != "" {
			req.Header.Set("Authorization", "Bearer "+bearerAuth)
		}
	case "apiKey":
		// Accept apiKey string from either basicAuth or apiKeyAuth for robustness
		apiKey := apiKeyAuth
		if apiKey == "" && basicAuth != "" {
			apiKey = basicAuth
		}
		if apiKey != "" {
			for _, part := range strings.Split(apiKey, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				// format passAs:name=value
				colonIdx := strings.Index(part, ":")
				eqIdx := strings.Index(part, "=")
				if colonIdx == -1 || eqIdx == -1 || eqIdx < colonIdx+2 {
					continue
				}
				passAs := strings.ToLower(strings.TrimSpace(part[:colonIdx]))
				name := strings.TrimSpace(part[colonIdx+1 : eqIdx])
				value := strings.TrimSpace(part[eqIdx+1:])
				switch passAs {
				case "header":
					req.Header.Set(name, value)
				case "query":
					// Update the query param in-place
					q := req.URL.Query()
					q.Set(name, value)
					req.URL.RawQuery = q.Encode()
				case "cookie":
					// Set the cookie header directly for test visibility
					existing := req.Header.Get("Cookie")
					if existing != "" {
						req.Header.Set("Cookie", existing+"; "+name+"="+value)
					} else {
						req.Header.Set("Cookie", name+"="+value)
					}
				}
			}
		}
	}
}

// CreateMCPToolHandler returns a ToolHandlerFunc that builds and sends HTTP requests for a given endpoint.
// It handles path, query, header, and body parameters, as well as security and custom headers.
func CreateMCPToolHandler(
	reqPathParam []string,
	reqQueryParam []string,
	reqURL string,
	reqBody map[string]string,
	reqMethod string,
	reqHeader []string,
	apiCfg models.ApiConfig,
) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		currentReqURL := reqURL
		for _, paramName := range reqPathParam {
			param, ok := request.Params.Arguments[paramName].(string)
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("[Error] missing or invalid Path Parameter: %s", paramName)), nil
			}
			currentReqURL = strings.Replace(currentReqURL, fmt.Sprintf("{%s}", paramName), param, 1)
		}
		// query param
		if len(reqQueryParam) > 0 {
			u, err := url.Parse(currentReqURL)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("[Error] failed to parse URL: %v", err)), nil
			}
			q := u.Query()
			for _, name := range reqQueryParam {
				val, ok := request.Params.Arguments[name].(string)
				if !ok {
					return mcp.NewToolResultError(fmt.Sprintf("[Error] missing or invalid Query Parameter: %s", name)), nil
				}
				q.Set(name, val)
			}
			u.RawQuery = q.Encode()
			currentReqURL = u.String()
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
		fmt.Printf("Request  : %s %s\n", strings.ToUpper(reqMethod), currentReqURL)
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
		req.Header.Set("Content-Type", "application/json")
		// request security
		setRequestSecurity(req, apiCfg.Security, apiCfg.BasicAuth, apiCfg.ApiKeyAuth, apiCfg.BearerAuth)
		// set custom headers from ApiConfig.Headers (format: name1=value1,name2=value2)
		if apiCfg.Headers != "" {
			for _, pair := range strings.Split(apiCfg.Headers, ",") {
				if pair = strings.TrimSpace(pair); pair == "" {
					continue
				}
				if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
					if key := strings.TrimSpace(kv[0]); key != "" {
						req.Header.Add(key, strings.TrimSpace(kv[1]))
					}
				}
			}
		}
		// headers from sse
		sseHeadersValue := ctx.Value(sseHeadersKey)
		if sseHeadersValue != nil {
			if sseHeaders, ok := sseHeadersValue.(map[string]string); ok {
				for k, v := range sseHeaders {
					req.Header.Set(k, v)
				}
			}
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
		fmt.Printf("Response : %s\n", string(body))
		return mcp.NewToolResultText(string(body)), nil
	}
}
