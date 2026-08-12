package router

import (
	"os"
	"strings"
	"testing"

	"aiapigateway/pkg/config"
)

var cfg = config.Config{
	PORT:          "8080",
	TargetURL:     "https://api.anthropic.com/v1/messages",
	UserAPIkey:    os.Getenv("ANTHROPIC_API_KEY"),
	StubMessage:   4,
	TruncateHead:  50,
	TruncateTail:  100,
	CheapModel:    "claude-3-5-haiku-20241022",
	RetryAttempts: 3,
}

func TestRouteModel(t *testing.T) {
	longPadding := strings.Repeat("This is extra conversational padding to exceed the four hundred character limit set in the router length veto check. ", 5)

	tests := []struct {
		name          string
		initialModel  string
		prompt        string
		expectedModel string
	}{
		{
			name:          "Trivial request downgrades to Haiku",
			initialModel:  "claude-3-5-sonnet-20241022",
			prompt:        "write a commit message for adding auth routes",
			expectedModel: "claude-3-5-haiku-20241022",
		},
		{
			name:          "Complex query stays Sonnet",
			initialModel:  "claude-3-5-sonnet-20241022",
			prompt:        "Why does this goroutine deadlock on channel receive?",
			expectedModel: "claude-3-5-sonnet-20241022",
		},

		{
			name:          "Edge Case: Strips markdown code block to ignore veto words (bug, panic)",
			initialModel:  "claude-3-5-sonnet-20241022",
			prompt:        "Format this code:\n```go\n// fixed a bug causing a panic\nfunc main() {}\n```",
			expectedModel: "claude-3-5-haiku-20241022", // safely downgrades because code was stripped
		},
		{
			name:          "Edge Case: Strips XML block to ignore veto words",
			initialModel:  "claude-3-5-sonnet-20241022",
			prompt:        "write a commit message for these changes:\n<file>\nfunc parse() error {\n return nil \n}\n</file>",
			expectedModel: "claude-3-5-haiku-20241022", // safely downgrades because <file> was stripped
		},

		{
			name:          "Edge Case: Unfenced code triggers veto (Regex Blindspot)",
			initialModel:  "claude-3-5-sonnet-20241022",
			prompt:        "format this code:\n    func test() {\n        return error\n    }",
			expectedModel: "claude-3-5-sonnet-20241022", // stays Sonnet because 4-space indent isn't stripped and "error" triggers Phase 2
		},
		{
			name:          "Edge Case: Instruction length exceeds 400 chars (Veto)",
			initialModel:  "claude-3-5-sonnet-20241022",
			prompt:        "Format this code. " + longPadding,
			expectedModel: "claude-3-5-sonnet-20241022", // stays Sonnet because the instruction is too long
		},
		{
			name:          "Edge Case: Semantic Drift (False Downgrade)",
			initialModel:  "claude-3-5-sonnet-20241022",
			prompt:        "Translate this monolithic SQL schema into a distributed GraphQL microservice.",
			expectedModel: "claude-3-5-haiku-20241022", // Downgrades because "Translate" hits phase 1, and no phase 2 words are present
		},
		{
			name:          "Edge Case: Mixed case and punctuation",
			initialModel:  "claude-3-5-sonnet-20241022",
			prompt:        "wRiTe a ComMit mEsSage!!!",
			expectedModel: "claude-3-5-haiku-20241022",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := config.Req{
				Model: tt.initialModel,
				Messages: []config.Message{
					{Role: "user", Content: tt.prompt},
				},
			}

			RouteModel(&req, &cfg)

			if req.Model != tt.expectedModel {
				t.Errorf("\nFAILED: %s\nGOT:      %s\nEXPECTED: %s", tt.name, req.Model, tt.expectedModel)
			}
		})
	}
}
