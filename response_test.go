package typesafe

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeAnswerRequiresCompleteMaps(t *testing.T) {
	choice := wireQuestion{Type: "choice", Criteria: map[string]any{"yes": nil, "no": nil}}
	score := wireQuestion{Type: "score", Criteria: []any{"low", "high"}}
	tests := []struct {
		name      string
		question  wireQuestion
		body      string
		wantError string
	}{
		{"missing choice label", choice, `{"type":"choice","choice":"yes","confidence":1,"probabilities":{"yes":1}}`, `missing label "no"`},
		{"null choice map", choice, `{"type":"choice","choice":"yes","confidence":1,"probabilities":null}`, "missing label"},
		{"explicit zero choice", choice, `{"type":"choice","choice":"yes","confidence":1,"probabilities":{"yes":1,"no":0}}`, ""},
		{"missing score probability", score, `{"type":"score","score":0,"confidence":1,"probabilities":{"0":1},"legend":{"0":"low","1":"high"}}`, "probabilities: missing level 1"},
		{"missing score legend", score, `{"type":"score","score":0,"confidence":1,"probabilities":{"0":1,"1":0},"legend":{"0":"low"}}`, "legend: missing level 1"},
		{"explicit zero score", score, `{"type":"score","score":0,"confidence":1,"probabilities":{"0":1,"1":0},"legend":{"0":"low","1":"high"}}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeAnswer(json.RawMessage(tt.body), tt.question)
			if tt.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("want error containing %q, got %v", tt.wantError, err)
			}
		})
	}
}
