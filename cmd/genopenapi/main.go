// Command genopenapi prints the OpenAPI spec to stdout.
// Sourced from the embedded YAML so docs and the binary cannot drift.
package main

import (
	"fmt"
	"os"

	"github.com/yawo/onefacture/internal/gateway/openapi"
)

func main() {
	if _, err := os.Stdout.Write(openapi.Spec()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
