// Package openapi serves the OpenAPI 3.1 spec and a Scalar UI for it.
package openapi

import (
	_ "embed"
	"fmt"
	"net/http"
)

//go:embed spec.yaml
var specYAML []byte

// SpecHandler serves the OpenAPI spec as YAML (content negotiation kept simple).
func SpecHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(specYAML)
	}
}

// ScalarHandler renders Scalar UI pointing to /openapi.json.
func ScalarHandler(baseURL string) http.HandlerFunc {
	html := fmt.Sprintf(`<!doctype html>
<html>
<head>
  <title>onefacture API</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
  <script id="api-reference" data-url="%s/openapi.json"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`, baseURL)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}
}
