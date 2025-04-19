package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"strings"

	mcpserver "github.com/danishjsheikh/swagger-mcp/app/mcp-server"
	"github.com/danishjsheikh/swagger-mcp/app/swagger"
)

func main() {
	specUrl := flag.String("specUrl", "", "URL of the Swagger JSON specification")
	sseMode := flag.Bool("sse", false, "Run in SSE mode instead of stdio mode")
	baseUrl := flag.String("baseUrl", "http://localhost", "Base URL for the SSE server")
	port := flag.Int("port", 8080, "Port for the SSE server")
	flag.Parse()

	// Validate spec
	if *specUrl == "" {
		log.Fatal("Please provide the Swagger JSON URL using the --specUrl flag")
	}
	_, err := url.ParseRequestURI(*specUrl)
	if err != nil {
		log.Fatalf("Invalid spec URL: %v", err)
	}

	// Validate baseUrl
	if !strings.HasPrefix(*baseUrl, "http://") && !strings.HasPrefix(*baseUrl, "https://") {
		log.Fatal("baseUrl must start with http:// or https://")
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

	fmt.Printf("Starting server with specUrl: %s, SSE mode: %v, Base URL: %s, Port: %d\n", *specUrl, *sseMode, *baseUrl, *port)
	mcpserver.CreateServer(swaggerSpec, *sseMode, *baseUrl, *port)
}
