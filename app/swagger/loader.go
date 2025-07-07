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

const DefaultMaxSpecSize = 10 * 1024 * 1024 // 10 MB

var maxSpecSize = -1 // -1 means not initialized

// SetMaxSpecSize allows users to override the maximum allowed spec size (in bytes)
func SetMaxSpecSize(n int) {
	maxSpecSize = n
}

// GetMaxSpecSize returns the current max spec size (in bytes)
func GetMaxSpecSize() int {
	if maxSpecSize > 0 {
		return maxSpecSize
	}
	// Check env var if not set
	if val := os.Getenv("SWAGGER_MCP_MAX_SPEC_SIZE"); val != "" {
		if n, err := parseSize(val); err == nil {
			return n
		}
	}
	return DefaultMaxSpecSize
}

// parseSize parses a string as an int (bytes), supports e.g. "10MB", "1048576"
func parseSize(s string) (int, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	mult := 1
	switch {
	case strings.HasSuffix(s, "KB"):
		mult = 1024
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "MB"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "GB"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	}
	n, err := fmt.Sscanf(s, "%d", &mult)
	if err == nil && n == 1 {
		return mult, nil
	}
	// fallback: try ParseInt
	var val int
	_, err = fmt.Sscanf(s, "%d", &val)
	if err == nil {
		return val * mult, nil
	}
	return 0, fmt.Errorf("invalid size: %s", s)
}

func LoadSwagger(specUrl string) (models.SwaggerSpec, error) {
	var body []byte
	maxSize := GetMaxSpecSize()

	if strings.HasPrefix(specUrl, "file://") {
		filePath := strings.TrimPrefix(specUrl, "file://")
		f, err := os.Open(filePath)
		if err != nil {
			return models.SwaggerSpec{}, fmt.Errorf("error reading file: %v", err)
		}
		defer f.Close()
		body, err = io.ReadAll(io.LimitReader(f, int64(maxSize)+1))
		if err != nil {
			return models.SwaggerSpec{}, fmt.Errorf("error reading file: %v", err)
		}
		if len(body) > maxSize {
			return models.SwaggerSpec{}, fmt.Errorf("spec file too large (max %d bytes)", maxSize)
		}
	} else if strings.Contains(specUrl, "://") {
		resp, err := http.Get(specUrl)
		if err != nil {
			return models.SwaggerSpec{}, fmt.Errorf("error getting spec: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return models.SwaggerSpec{}, fmt.Errorf("error getting spec: status %d", resp.StatusCode)
		}

		body, err = io.ReadAll(io.LimitReader(resp.Body, int64(maxSize)+1))
		if err != nil {
			return models.SwaggerSpec{}, fmt.Errorf("error reading spec: %v", err)
		}
		if len(body) > maxSize {
			return models.SwaggerSpec{}, fmt.Errorf("spec file too large (max %d bytes)", maxSize)
		}
	} else {
		// treat as local file path
		f, err := os.Open(specUrl)
		if err != nil {
			return models.SwaggerSpec{}, fmt.Errorf("error reading file: %v", err)
		}
		defer f.Close()
		body, err = io.ReadAll(io.LimitReader(f, int64(maxSize)+1))
		if err != nil {
			return models.SwaggerSpec{}, fmt.Errorf("error reading file: %v", err)
		}
		if len(body) > maxSize {
			return models.SwaggerSpec{}, fmt.Errorf("spec file too large (max %d bytes)", maxSize)
		}
	}

	var swaggerSpec models.SwaggerSpec
	if err := json.Unmarshal(body, &swaggerSpec); err != nil {
		return models.SwaggerSpec{}, fmt.Errorf("error parsing JSON: %v", err.Error())
	}
	return swaggerSpec, nil
}
