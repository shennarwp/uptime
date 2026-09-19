package api

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOpenAPIContractContainsImplementedEndpoints(t *testing.T) {
	data, err := os.ReadFile("swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	required := map[string][]string{
		"/api/v1/targets":     {"get", "post"},
		"/api/v1/target/{id}": {"put", "delete"},
		"/api/v1/auth/verify": {"post"},
	}
	for path, methods := range required {
		for _, method := range methods {
			if _, ok := document.Paths[path][method]; !ok {
				t.Errorf("OpenAPI contract is missing %s %s", method, path)
			}
		}
	}
}
