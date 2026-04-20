package api

import (
	"os"
	"strings"
	"testing"
)

func TestOpenAPIHasCorePaths(t *testing.T) {
	b, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, p := range []string{
		"/healthz",
		"/v1/areas/changi",
		"/v1/stores",
		"/v1/stores/{id}",
		"/v1/stores/{id}/status-reports",
	} {
		if !strings.Contains(s, p) {
			t.Fatalf("openapi missing %q", p)
		}
	}
}
