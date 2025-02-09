package swagger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/danishjsheikh/swagger-mcp/app/models"
)

func LoadSwagger() (models.SwaggerSpec, error) {
	if len(os.Args) < 2 {
		return models.SwaggerSpec{}, fmt.Errorf("usage: go run main.go <swagger_json_url>")
	}
	reqURL := os.Args[1]
	resp, err := http.Get(reqURL)
	if err != nil {
		return models.SwaggerSpec{}, fmt.Errorf("error Getting swagger/doc.json, %v", err.Error())

	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.SwaggerSpec{}, fmt.Errorf("error reading swagger/doc.json, %v", err.Error())

	}
	var swaggerSpec models.SwaggerSpec
	if err := json.Unmarshal(body, &swaggerSpec); err != nil {
		return models.SwaggerSpec{}, fmt.Errorf("error parsing JSON:, %v", err.Error())
	}
	return swaggerSpec, nil
}
