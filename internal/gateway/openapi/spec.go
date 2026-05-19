package openapi

// Spec returns the raw OpenAPI YAML.
func Spec() []byte {
	out := make([]byte, len(specYAML))
	copy(out, specYAML)
	return out
}
