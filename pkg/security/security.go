package security

import (
	"aiapigateway/pkg/config"
	"regexp"
)

var (
	awsKeyRegex     = regexp.MustCompile(`\b(AKIA|ASIA)[0-9A-Z]{16}\b`)
	awsSecretRegex  = regexp.MustCompile(`(?i)\baws_secret_access_key\s*=\s*['"]?[A-Za-z0-9/+=]{40}['"]?\b`)
	jwtRegex        = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
	bearerRegex     = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9\-._~+/]+=*\b`)
	genericKeyRegex = regexp.MustCompile(`(?i)\b(api[_-]?key|secret[_-]?key|auth[_-]?token)\s*[:=]\s*['"]?[A-Za-z0-9_\-]{16,}['"]?\b`)

	githubTokenRegex  = regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{30,255}\b`)
	stripeKeyRegex    = regexp.MustCompile(`\b(sk|pk)_(test|live)_[0-9a-zA-Z]{24,99}\b`)
	slackTokenRegex   = regexp.MustCompile(`xox[baprs]-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*`)
	openAIKeyRegex    = regexp.MustCompile(`\bsk-proj-[A-Za-z0-9_-]{32,}\b|\bsk-[A-Za-z0-9]{48}\b`)
	anthropicKeyRegex = regexp.MustCompile(`\bsk-ant-api03-[A-Za-z0-9_-]{80,}\b`)
	dbUrlRegex        = regexp.MustCompile(`(?i)\b(postgres|postgresql|mysql|mongodb(\+srv)?):\/\/[^:]+:[^@]+@[^:\/\s]+:\d+\/[^\s\?"']+\b`)
	sshKeyRegex       = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]+KEY-----.*?-----END [A-Z ]+KEY-----`)
)

func StripSecrets(text string) string {
	s := awsKeyRegex.ReplaceAllString(text, "[REDACTED_AWS_KEY]")
	s = awsSecretRegex.ReplaceAllString(s, "aws_secret_access_key=[REDACTED_SECRET]")
	s = jwtRegex.ReplaceAllString(s, "[REDACTED_JWT_TOKEN]")
	s = bearerRegex.ReplaceAllString(s, "Bearer [REDACTED_BEARER_TOKEN]")
	s = genericKeyRegex.ReplaceAllString(s, "$1=[REDACTED_KEY]")

	s = githubTokenRegex.ReplaceAllString(s, "[REDACTED_GITHUB_TOKEN]")
	s = stripeKeyRegex.ReplaceAllString(s, "[REDACTED_STRIPE_KEY]")
	s = slackTokenRegex.ReplaceAllString(s, "[REDACTED_SLACK_TOKEN]")
	s = openAIKeyRegex.ReplaceAllString(s, "[REDACTED_OPENAI_KEY]")
	s = anthropicKeyRegex.ReplaceAllString(s, "[REDACTED_ANTHROPIC_KEY]")
	s = dbUrlRegex.ReplaceAllString(s, "[REDACTED_DB_CONNECTION_STRING]")
	s = sshKeyRegex.ReplaceAllString(s, "[REDACTED_PRIVATE_SSH_KEY]")

	return s
}

func RedactPayload(req *config.Req) {
	for i := range req.Messages {
		msg := &req.Messages[i]
		switch content := msg.Content.(type) {
		case string:
			msg.Content = StripSecrets(content)
		case []config.ContentBlock:
			for j := range content {
				if content[j].Type == "text" {
					content[j].Text = StripSecrets(content[j].Text)
				}
			}
		}
	}
}
