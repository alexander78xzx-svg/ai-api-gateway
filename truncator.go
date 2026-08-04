package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

func truncateLogs(req *Req) error {
	for i := range req.Messages {

		blocks, ok := req.Messages[i].Content.([]ContentBlock) // checking if conent is a slice of contentblocks
		if !ok || len(blocks) == 0 {
			continue
		}

		// we search for the tool output made by the user role
		if req.Messages[i].Role == "user" {

			// we check if the message is older than n turns
			if len(req.Messages) > 4 && i < len(req.Messages)-4 {

				for j := range blocks {
					if blocks[j].Type != "tool_result" {
						continue
					}

					// check if tool result is a string

					switch content := blocks[j].Content.(type) {
					case string:
						if len(strings.TrimSpace(content)) == 0 {
							continue
						}

						blocks[j].Content = fmt.Sprintf("Stubbed output, %d chars removed", len(content))

					case []ContentBlock:
						var logLen int

						for _, innerBlock := range content {
							if innerBlock.Type == "text" {
								logLen += len(innerBlock.Text)
							}
						}

						// fallback
						if logLen == 0 && len(content) > 0 {
							if data, err := json.Marshal(content); err == nil {
								logLen = len(data)
							}
						}
						if logLen > 0 {
							blocks[j].Content = fmt.Sprintf("Stubbed output, %d chars removed", logLen)
						}
					}
				}
			} else {
				// message isnt older than n turns (needs to be preserved)

				// each parsed block:
				for j := range blocks {

					if blocks[j].Type != "tool_result" {
						continue
					}

					// check if tool result is a string

					switch content := blocks[j].Content.(type) {
					case string:
						if len(strings.TrimSpace(content)) == 0 {
							continue
						}

						parts := strings.Split(content, "\n")

						if len(parts) > cfg.truncateHead+cfg.truncateTail {
							head := parts[:cfg.truncateHead]
							tail := parts[len(parts)-cfg.truncateTail:]
							note := fmt.Sprintf("[%d lines truncated]", len(parts)-(cfg.truncateHead+cfg.truncateTail))

							var temp []string
							temp = append(temp, head...)
							temp = append(temp, note)
							temp = append(temp, tail...)

							blocks[j].Content = strings.Join(temp, "\n")
						}

					case []ContentBlock:
						continue
					}

				}
			}
		}

	}
	return nil
}
