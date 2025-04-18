package swagger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/danishjsheikh/swagger-mcp/app/models"
)

func LoadSwagger(docJsonUrl string) (models.SwaggerSpec, error) {
	resp, err := http.Get(docJsonUrl)
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
