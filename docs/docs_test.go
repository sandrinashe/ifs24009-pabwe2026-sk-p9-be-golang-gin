package docs

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSwaggerJSON(t *testing.T) {
	b, err := GetSwaggerJSON("example.com", "https")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	paths := doc["paths"].(map[string]any)
	if len(paths) < 15 || !strings.Contains(string(b), "https://example.com") || !strings.Contains(string(b), "bearerAuth") {
		t.Fatal("dokumen tidak lengkap")
	}
	if !strings.Contains(SwaggerUIHTML, "/api.json") || contains("abc", "zz") {
		t.Fatal("html/contains")
	}
}
