package main

import (
	"github.com/danishjsheikh/swagger-mcp/app/swagger"
)

func main() {
	swaggerSpec, err := swagger.LoadSwagger()
	if err != nil {
		panic(err)
	}
	swagger.ExtractSwagger(swaggerSpec)
}
