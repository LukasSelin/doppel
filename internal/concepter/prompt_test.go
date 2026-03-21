package concepter

import (
	"encoding/json"
	"testing"
)

func TestFlexStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "normal array",
			input: `["context.Context","string"]`,
			want:  []string{"context.Context", "string"},
		},
		{
			name:  "single string",
			input: `"context.Context"`,
			want:  []string{"context.Context"},
		},
		{
			name:  "empty string",
			input: `""`,
			want:  nil,
		},
		{
			name:  "null",
			input: `null`,
			want:  nil,
		},
		{
			name:  "empty array",
			input: `[]`,
			want:  []string{},
		},
		{
			name:  "array of objects with type key",
			input: `[{"type":"context.Context"},{"type":"string"}]`,
			want:  []string{"context.Context", "string"},
		},
		{
			name:  "array of objects with name key",
			input: `[{"name":"ctx","type":"context.Context"}]`,
			want:  []string{"context.Context"},
		},
		{
			name:  "object instead of array",
			input: `{"type":"context.Context"}`,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got flexStringSlice
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseConceptResponse(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantSummary string
		wantInputs  []string
		wantErr     bool
	}{
		{
			name: "clean JSON",
			raw: `{
				"summary": "fetches a user by ID",
				"inputs": ["context.Context", "string"],
				"outputs": ["User", "error"],
				"dependencies": ["database/sql"],
				"patterns": ["db_access"]
			}`,
			wantSummary: "fetches a user by ID",
			wantInputs:  []string{"context.Context", "string"},
		},
		{
			name: "markdown fenced JSON",
			raw: "```json\n{\"summary\":\"does something\",\"inputs\":[\"string\"]}\n```",
			wantSummary: "does something",
			wantInputs:  []string{"string"},
		},
		{
			name: "inputs as single string",
			raw:  `{"summary":"x","inputs":"string","outputs":[],"dependencies":[],"patterns":[]}`,
			wantSummary: "x",
			wantInputs:  []string{"string"},
		},
		{
			name: "inputs as array of objects",
			raw:  `{"summary":"y","inputs":[{"type":"int"},{"type":"bool"}],"outputs":[],"dependencies":[],"patterns":[]}`,
			wantSummary: "y",
			wantInputs:  []string{"int", "bool"},
		},
		{
			name:    "no JSON in response",
			raw:     "I cannot analyze this function.",
			wantErr: true,
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseConceptResponse(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Summary != tt.wantSummary {
				t.Errorf("Summary = %q, want %q", got.Summary, tt.wantSummary)
			}
			if len(got.Inputs) != len(tt.wantInputs) {
				t.Fatalf("Inputs = %v, want %v", got.Inputs, tt.wantInputs)
			}
			for i := range got.Inputs {
				if got.Inputs[i] != tt.wantInputs[i] {
					t.Errorf("Inputs[%d] = %q, want %q", i, got.Inputs[i], tt.wantInputs[i])
				}
			}
		})
	}
}
