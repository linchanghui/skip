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
		"/v1/tasks",
		"/v1/tasks/{id}",
		"/v1/tasks/{id}/cancel",
		"/v1/runners/apply",
		"/v1/runners/{id}/availability",
		"/v1/tasks/{id}/accept",
		"/v1/tasks/{id}/arrive",
		"/v1/tasks/{id}/complete",
		"/v1/tasks/{id}/proofs",
		"/v1/stores/{id}/queue-reports",
		"/v1/stores/{id}/queue-signal",
		"/v1/metrics/summary",
		"/v1/ops/tasks/{id}/assign",
		"/v1/ops/queue-reports/{id}/hide",
	} {
		if !strings.Contains(s, p) {
			t.Fatalf("openapi missing %q", p)
		}
	}
}
