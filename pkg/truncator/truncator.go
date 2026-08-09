package truncator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"aiapigateway/pkg/config"
)

var (
	codeKeywordRegex = regexp.MustCompile(`(?m)^\s*(import|export|function|func|const|let|var|class|def|return|package|if|else|struct|interface|type)\b`)
	codeSyntaxRegex  = regexp.MustCompile(`(\{|\}|=>|:=|->|;\s*$|\[\])`)

	errorKeywordRegex = regexp.MustCompile(`(?i)\b(error|exception|fail|failed|fatal|panic|traceback|err|undefined|cannot|unable)\b|:\d+:\d+`)
	ansiRegexp        = regexp.MustCompile(
		`\x1b(?:` +
			`\[[0-?]*[ -/]*[@-~]|` +
			`\][^\x07]*(?:\x07|\x1b\\)|` +
			`[@-Z\\-_]` +
			`)`,
	)
	multiSpaceRegex = regexp.MustCompile(` {2,}`)
)

func extractString(val any) (string, bool) {
	s, ok := val.(string)
	return strings.TrimSpace(s), ok
}

func IsJSON(input any) bool {
	s, ok := extractString(input)
	if !ok || s == "" {
		return false
	}
	if !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[") {
		return false
	}
	return json.Valid([]byte(s))
}

func IsCode(input any) bool {
	s, ok := extractString(input)
	if !ok || s == "" {
		return false
	}
	if strings.HasPrefix(s, "```") {
		return true
	}

	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return false
	}

	keywordMatches := 0
	syntaxMatches := 0

	for _, line := range lines {
		if codeKeywordRegex.MatchString(line) {
			keywordMatches++
		}
		if codeSyntaxRegex.MatchString(line) && !strings.HasPrefix(strings.TrimSpace(line), "[") {
			syntaxMatches++
		}
	}

	totalLines := len(lines)
	return keywordMatches > 0 || (float64(syntaxMatches)/float64(totalLines)) > 0.4
}

func ShouldSkipTruncation(content any) bool {
	return IsJSON(content) || IsCode(content)
}

// main
func TruncateLogs(req *config.Req, cfg *config.Config) {
	if len(req.Messages) == 0 {
		return
	}

	// only last message
	lastIdx := len(req.Messages) - 1
	msg := &req.Messages[lastIdx]

	switch content := msg.Content.(type) {
	case string:
		if !ShouldSkipTruncation(content) {
			msg.Content = applyTruncationPipeline(content, cfg)
		}

	case []config.ContentBlock:

		for i := range content {
			if blockText, ok := content[i].Content.(string); ok {
				if !ShouldSkipTruncation(blockText) {
					content[i].Content = applyTruncationPipeline(blockText, cfg)
				}
			}
		}
	}
}

// helper funcs for pipeline

func cleanCharacters(s string) string {
	if s == "" {
		return ""
	}

	// strip ansii patterns
	s = ansiRegexp.ReplaceAllString(s, "")

	// normalize \r\n to \n
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	var b strings.Builder
	b.Grow(len(s))

	lines := strings.Split(s, "\n")
	prevBlank := false
	writtenLines := 0
	var lastWrittenLine string

	for _, line := range lines {
		var lineBuilder strings.Builder
		lineBuilder.Grow(len(line))

		for _, r := range line {
			if r == '\b' || unicode.Is(unicode.Braille, r) {
				continue
			}
			switch r {
			case '◐', '◓', '◑', '◒', '◴', '◷', '◶', '◵', '○', '●',
				'◜', '◝', '◞', '◟', '◠', '◡', '◢', '◣', '◤', '◥',
				'▖', '▘', '▝', '▗', '▌', '▀', '▄', '█', '■', '□':
				continue
			}

			if unicode.IsPrint(r) || r == '\t' {
				lineBuilder.WriteRune(r)
			}
		}

		cleanedLine := strings.Trim(lineBuilder.String(), " \t\r")

		cleanedLine = multiSpaceRegex.ReplaceAllString(cleanedLine, " ")

		isBlank := cleanedLine == ""

		if isBlank {
			if prevBlank {
				continue
			}
			prevBlank = true
		} else {
			if cleanedLine == lastWrittenLine {
				continue
			}
			prevBlank = false
			lastWrittenLine = cleanedLine
		}

		if writtenLines > 0 {
			b.WriteRune('\n')
		}
		b.WriteString(cleanedLine)
		writtenLines++
	}

	return strings.TrimRight(b.String(), "\n\t ")
}

func headTailTruncate(s string, headLines, tailLines, windowSize int) string {
	lines := strings.Split(s, "\n")
	total := len(lines)

	if total <= (headLines + tailLines) {
		return s
	}

	keep := make(map[int]bool)

	for i := 0; i < headLines && i < total; i++ {
		keep[i] = true
	}

	for i := total - tailLines; i < total; i++ {
		if i >= 0 {
			keep[i] = true
		}
	}

	for i := headLines; i < total-tailLines; i++ {
		if errorKeywordRegex.MatchString(lines[i]) {

			start := i - windowSize
			if start < 0 {
				start = 0
			}
			end := i + windowSize
			if end >= total {
				end = total - 1
			}

			for j := start; j <= end; j++ {
				keep[j] = true
			}
		}
	}

	var b strings.Builder
	omitting := false
	omittedCount := 0

	for i := 0; i < total; i++ {
		if keep[i] {
			if omitting {
				b.WriteString(fmt.Sprintf("\n[... %d lines omitted]\n", omittedCount))
				omitting = false
				omittedCount = 0
			}
			if b.Len() > 0 {
				b.WriteRune('\n')
			}
			b.WriteString(lines[i])
		} else {
			omitting = true
			omittedCount++
		}
	}

	if omitting {
		b.WriteString(fmt.Sprintf("\n[... %d lines omitted]", omittedCount))
	}

	return b.String()
}

// pipeline :
func applyTruncationPipeline(logText string, cfg *config.Config) string {

	originalLen := len(logText)
	if originalLen == 0 {
		return logText
	}

	// tier 1:
	logText = cleanCharacters(logText)

	// tier 2:

	logText = headTailTruncate(logText, cfg.TruncateHead, cfg.TruncateTail, 3)

	return logText
}
