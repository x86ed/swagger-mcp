package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	mcpserver "github.com/danishjsheikh/swagger-mcp/app/mcp-server"
	"github.com/danishjsheikh/swagger-mcp/app/swagger"
)

func main() {
	specUrl := flag.String("specUrl", "", "URL of the Swagger JSON specification")
	sseMode := flag.Bool("sse", false, "Run in SSE mode instead of stdio mode")
	sseUrl := flag.String("sseUrl", "http://localhost:8080", "URL for the SSE server")
	baseUrl := flag.String("baseUrl", "", "Base URL for API requests")
	port := flag.Int("port", 8080, "Port for the SSE server")
	includePaths := flag.String("includePaths", "", "Comma-separated list of paths or regex to include")
	excludePaths := flag.String("excludePaths", "", "Comma-separated list of paths or regex to exclude")
	includeMethods := flag.String("includeMethods", "", "Comma-separated list of HTTP methods to include")
	excludeMethods := flag.String("excludeMethods", "", "Comma-separated list of HTTP methods to exclude")
	flag.Parse()

	// Validate spec
	if *specUrl == "" {
		log.Fatal("Please provide the Swagger JSON URL or file path using the --specUrl flag")
	}

	if strings.HasPrefix(*specUrl, "http://") || strings.HasPrefix(*specUrl, "https://") {
		_, err := url.ParseRequestURI(*specUrl)
		if err != nil {
			log.Fatalf("Invalid spec URL: %v", err)
		}
	} else if strings.HasPrefix(*specUrl, "file://") {
		filePath := strings.TrimPrefix(*specUrl, "file://")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Fatalf("Spec file does not exist: %v", err)
		}
	} else {
		log.Fatal("Invalid specUrl format. Must be a valid HTTP URL or file:// path")
	}

	// Validate sseUrl
	if !strings.HasPrefix(*sseUrl, "http://") && !strings.HasPrefix(*sseUrl, "https://") {
		log.Fatal("sseUrl must start with http:// or https://")
	}

	// Validate baseUrl
	if *baseUrl != "" {
		if !strings.HasPrefix(*baseUrl, "http://") && !strings.HasPrefix(*baseUrl, "https://") {
			log.Fatal("baseUrl must start with http:// or https://")
		}
	}

	// Validate port
	if *port < 1 || *port > 65535 {
		log.Fatal("Port must be between 1 and 65535")
	}

	swaggerSpec, err := swagger.LoadSwagger(*specUrl)
	if err != nil {
		log.Fatalf("Failed to load Swagger spec: %v", err)
	}
	swagger.ExtractSwagger(swaggerSpec)

	fmt.Printf("Starting server with specUrl: %s, SSE mode: %v, SSE URL: %s, Base URL: %s, Port: %d, Include Paths: %s, Exclude Paths: %s, Include Methods: %s, Exclude Methods: %s\n",
		*specUrl, *sseMode, *sseUrl, *baseUrl, *port, *includePaths, *excludePaths, *includeMethods, *excludeMethods)
	mcpserver.CreateServer(swaggerSpec, *sseMode, *sseUrl, *baseUrl, *port, *includePaths, *excludePaths, *includeMethods, *excludeMethods)
}
