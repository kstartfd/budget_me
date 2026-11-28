package commands

import (
	"encoding/json"
	"testing"
)

func TestDiagnosticGenerationCannotReturnAuthorityOrIdentifiers(t *testing.T) {
	for _, body := range []string{
		`{"questions":[],"level":"B2"}`,
		`{"questions":[{"prompt":"Select one","knowledgeItemKey":"invented"}]}`,
		`{"questions":[{"prompt":"Select one","difficultyBand":"C1"}]}`,
	} {
		var out diagnosticContent
		if err := json.Unmarshal([]byte(body), &out); err == nil {
			t.Fatalf("accepted out-of-contract data: %s", body)
		}
	}
}
