package router

import (
	"regexp"
	"strings"

	"aiapigateway/pkg/config"
)

var (
	codeBlockRegex = regexp.MustCompile(`(?s)\x60\x60\x60.*?\x60\x60\x60`)
	xmlBlockRegex  = regexp.MustCompile(`(?s)<[a-zA-Z0-9_]+>.*?</[a-zA-Z0-9_]+>`)

	trivialRegex = regexp.MustCompile(`(?i)\b(commit message|docstrings?|comments?|jsdoc|format|lint|typos?|spelling|translate)\b`)

	complexityRegex = regexp.MustCompile(`(?i)\b(bug|panic|crash|leak|race|optimize|refactor|security|vulnerability|deadlock|segfault|why|how|explain|logic|architecture|review|rewrite|issue|error|fail|broken)\b`)
)

// strips code chunks to isolate the human/agent command. (for better context for the router)
func extractInstruction(prompt string) string {
	s := codeBlockRegex.ReplaceAllString(prompt, "")
	s = xmlBlockRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func RouteModel(req *config.Req, cfg *config.Config) {

	if req.Model == cfg.CheapModel || strings.Contains(req.Model, "haiku") {
		return
	}

	if len(req.Messages) == 0 {
		return
	}
	lastMsg := req.Messages[len(req.Messages)-1]

	var fullPrompt string
	switch content := lastMsg.Content.(type) {
	case string:
		fullPrompt = content
	case []config.ContentBlock:
		for _, block := range content {
			if block.Type == "text" {
				fullPrompt += block.Text + "\n"
			}
		}
	}

	//  isolate human command from the attached code
	instruction := extractInstruction(fullPrompt)

	if len(instruction) > 400 {
		return
	}

	isTrivial := trivialRegex.MatchString(instruction)
	isComplex := complexityRegex.MatchString(instruction)

	if isTrivial && !isComplex {
		req.Model = cfg.CheapModel
	}
}
