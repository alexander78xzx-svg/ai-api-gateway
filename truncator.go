package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func truncateLogs(req *Req) error {
	for i := range req.Messages {
		// we search for the tool output made by the user role
		if req.Messages[i].Role == "user" && len(req.Messages[i].ParsedBlocks) > 0 {

			// we check if the message is older than n turns
			if len(req.Messages) > 4 && i < len(req.Messages)-4 {

				for j := range req.Messages[i].ParsedBlocks {
					if req.Messages[i].ParsedBlocks[j].Type != "tool_result" {
						continue
					}

					// check if tool result is a string
					trim := bytes.TrimSpace(req.Messages[i].ParsedBlocks[j].Content)
					if len(trim) == 0 {
						continue
					}
					switch trim[0] {
					case '"':
						var s string
						if err := json.Unmarshal(req.Messages[i].ParsedBlocks[j].Content, &s); err != nil {
							continue
						}

						s = fmt.Sprintf("Stubbed output, %d chars removed", len(s))
						t, err := json.Marshal(s)
						if err != nil {
							return err
						}

						req.Messages[i].ParsedBlocks[j].Content = json.RawMessage(t)
					case '[':
						var block []map[string]any
						err := json.Unmarshal(trim, &block)
						if err != nil {

							// fallback, if the content is other that strings, we just count raw json bytes
							logLen := len(trim)
							s := fmt.Sprintf("Stubbed output, %d chars removed", logLen)
							t, _ := json.Marshal(s)
							req.Messages[i].ParsedBlocks[j].Content = json.RawMessage(t)
							continue
						}
						var logLen int
						for _, b := range block {
							if text, ok := b["text"].(string); ok {
								logLen += len(text)
							}
						}
						if logLen > 0 {
							s := fmt.Sprintf("Stubbed output, %d chars removed", logLen)
							t, err := json.Marshal(s)
							if err != nil {
								return err

							}

							req.Messages[i].ParsedBlocks[j].Content = json.RawMessage(t)
						}
					}
				}
			} else {
				// message isnt older than n turns (needs to be preserved)

				// each parsed block:
				for j := range req.Messages[i].ParsedBlocks {

					if req.Messages[i].ParsedBlocks[j].Type != "tool_result" {
						continue
					}

					// check if tool result is a string
					trim := bytes.TrimSpace(req.Messages[i].ParsedBlocks[j].Content)
					if len(trim) == 0 {
						continue
					}

					switch trim[0] {
					case '"':
						var s string
						if err := json.Unmarshal(req.Messages[i].ParsedBlocks[j].Content, &s); err != nil {
							continue
						}

						parts := strings.Split(s, "\n")

						// truncate and leave only first 50 and last 100 lines, if enough lines
						if len(parts) > 150 {

							var temp []string
							head := parts[:50]
							tail := parts[len(parts)-100:]
							text := fmt.Sprintf("[%d lines truncated]", len(parts)-150) // add a note

							temp = append(temp, head...)
							temp = append(temp, text)
							temp = append(temp, tail...)

							// readd an \\n at the end of each log
							truncated := strings.Join(temp, "\n")

							t, err := json.Marshal(truncated)
							if err != nil {
								return err
							}

							req.Messages[i].ParsedBlocks[j].Content = json.RawMessage(t)
						}
					case '[':
						continue
					}

				}
			}

			// we convert parsed blocks back to content
			p, err := json.Marshal(req.Messages[i].ParsedBlocks)
			if err != nil {
				return err
			}
			req.Messages[i].Content = json.RawMessage(p)
		}

	}
	return nil
}
