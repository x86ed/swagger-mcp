package swagger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/danishjsheikh/swagger-mcp/app/models"
)

func LoadSwagger(specUrl string) (models.SwaggerSpec, error) {
	var body []byte
	var err error

	if strings.HasPrefix(specUrl, "file://") {
		filePath := strings.TrimPrefix(specUrl, "file://")
		body, err = os.ReadFile(filePath)
		if err != nil {
			return models.SwaggerSpec{}, fmt.Errorf("error reading file: %v", err)
		}
	} else {
		resp, err := http.Get(specUrl)
		if err != nil {
			return models.SwaggerSpec{}, fmt.Errorf("error getting spec: %v", err)
		}
		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return models.SwaggerSpec{}, fmt.Errorf("error reading spec: %v", err)
		}
	}
	var swaggerSpec models.SwaggerSpec
	if err := json.Unmarshal(body, &swaggerSpec); err != nil {
		return models.SwaggerSpec{}, fmt.Errorf("error parsing JSON:, %v", err.Error())
	}
	return swaggerSpec, nil
}
