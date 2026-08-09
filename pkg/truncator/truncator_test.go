package truncator

import (
	"fmt"
	"strings"
	"testing"
)

// test tier 1
func TestCleanCharacters(t *testing.T) {

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Strip ANSI color codes",
			input:    "\x1b[32m[INFO]\x1b[0m Starting build...",
			expected: "[INFO] Starting build...",
		},
		{
			name:     "Normalize carriage returns and collapse duplicates",
			input:    "Fetching...\rFetching...\rFetching...\r\nDone!",
			expected: "Fetching...\nDone!",
		},
		{
			name:     "Remove spinner frames and braille icons",
			input:    "⠋ [1/3] Loading...\n⠙ [1/3] Loading...\n✔ Loaded!",
			expected: "[1/3] Loading...\n✔ Loaded!",
		},
		{
			name:     "Remove block progress bar symbols",
			input:    "50% [██████████          ] Building...",
			expected: "50% [ ] Building...",
		},
		{
			name:     "Collapse repeating blank lines",
			input:    "Line 1\n\n\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "Clean trailing whitespace from lines",
			input:    "Line with space   \nAnother line\t\t",
			expected: "Line with space\nAnother line",
		},
		{
			name:     "Empty input returns empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanCharacters(tt.input)
			if got != tt.expected {
				t.Errorf("\nFail: %s\nGot:      %q\nexpected: %q", tt.name, got, tt.expected)
			}
		})
	}
}

// benchmark tier 1
func BenchmarkCleanCharacters(b *testing.B) {
	rawLog := "\x1b[32m[INFO]\x1b[0m Starting build pipeline...\r\n" +
		"◐ Fetching packages...\r" +
		"◓ Fetching packages...\r" +
		"◒ Fetching packages...\r\n" +
		"⠋ [1/3] Downloading dependencies...   \n" +
		"⠙ [1/3] Downloading dependencies...\n" +
		"✔ Dependencies downloaded successfully.\x1b[2K\r\n" +
		"\n\n" +
		"\x1b[31m[ERROR]\x1b[0m Failed to compile src/main.go:42\n" +
		"   undefined: myUndefinedVar   \n" +
		"█ 50% [██████████          ] Building...\n\n\n"

	// Reset timer to measure only the function execution
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cleanCharacters(rawLog)
	}
}

// tier 2 testing
func TestHeadTailTruncate(t *testing.T) {
	tests := []struct {
		name      string
		headLines int
		tailLines int
		window    int
		buildLog  func() string
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:      "Short log under threshold is untouched",
			headLines: 10,
			tailLines: 10,
			window:    2,
			buildLog: func() string {
				return "Line 1\nLine 2\nLine 3"
			},
			checkFunc: func(t *testing.T, result string) {
				if strings.Contains(result, "omitted") {
					t.Errorf("Expected log to remain untouched, but found omission marker")
				}
			},
		},
		{
			name:      "Long clean log is truncated with omission marker",
			headLines: 5,
			tailLines: 5,
			window:    2,
			buildLog: func() string {
				lines := make([]string, 50)
				for i := 0; i < 50; i++ {
					lines[i] = fmt.Sprintf("Clean build line %d", i+1)
				}
				return strings.Join(lines, "\n")
			},
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "[... 40 lines omitted]") {
					t.Errorf("Expected exact omission marker for 40 lines, got:\n%s", result)
				}
				if !strings.HasPrefix(result, "Clean build line 1") {
					t.Errorf("Expected head line 1 at start")
				}
				if !strings.HasSuffix(result, "Clean build line 50") {
					t.Errorf("Expected tail line 50 at end")
				}
			},
		},
		{
			name:      "Middle error line and surrounding context window are preserved",
			headLines: 5,
			tailLines: 5,
			window:    2,
			buildLog: func() string {
				lines := make([]string, 100)
				for i := 0; i < 100; i++ {
					lines[i] = fmt.Sprintf("Line %d", i+1)
				}
				lines[49] = "FATAL: database connection refused at db.go:88"
				return strings.Join(lines, "\n")
			},
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "FATAL: database connection refused") {
					t.Errorf("Expected error line to be preserved")
				}
				// verify context window, window=2 means lines 48, 49, 50, 51, 52 should exist
				if !strings.Contains(result, "Line 48") || !strings.Contains(result, "Line 52") {
					t.Errorf("Expected error context window (Line 48 to 52) to be preserved")
				}
				// ensure there are 2 separate omission markers (before and after error window)
				omissionCount := strings.Count(result, "omitted")
				if omissionCount != 2 {
					t.Errorf("Expected 2 omission sections around error window, found %d", omissionCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.buildLog()
			got := headTailTruncate(input, tt.headLines, tt.tailLines, tt.window)
			tt.checkFunc(t, got)
		})
	}
}

// benchmark tier 2
func BenchmarkHeadTailTruncate(b *testing.B) {
	// 500-line test log payload
	lines := make([]string, 500)
	for i := 0; i < 500; i++ {
		lines[i] = fmt.Sprintf("Step %d: Executing test suite module...", i+1)
	}
	lines[250] = "FAIL: TestAuthHandler failed in handler_test.go:104"
	rawLog := strings.Join(lines, "\n")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = headTailTruncate(rawLog, 20, 50, 3)
	}
}
