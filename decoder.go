package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func decodeRequest(req *Req, r *http.Request) error {
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}

	for i := range req.Messages {
		msg := &req.Messages[i]
		if msg.Content == nil {
			continue
		}

		switch v := msg.Content.(type) {
		case string: // already ok
			continue

		case []any: // need to normalize

			bytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("message %d: failed to serialize content: %w", i, err)
			}

			var blocks []ContentBlock
			if err := json.Unmarshal(bytes, &blocks); err != nil {
				return fmt.Errorf("message %d: invalid content block array: %w", i, err)
			}

			// we unmarshaled message.content, now we check message.content.content

			for j := range blocks {
				block := &blocks[j]
				if block.Content == nil {
					continue
				}

				switch inner := block.Content.(type) {
				case string:
					continue

				case []any:
					innerBytes, err := json.Marshal(inner)
					if err != nil {
						return fmt.Errorf("message %d block %d: failed to serialize inner content: %w", i, j, err)
					}

					var innerBlocks []ContentBlock
					if err := json.Unmarshal(innerBytes, &innerBlocks); err != nil {
						return fmt.Errorf("message %d block %d: invalid inner content array: %w", i, j, err)
					}

					block.Content = innerBlocks

				default:
					return fmt.Errorf("message %d block %d: unsupported content type %T", i, j, inner)
				}
			}

			msg.Content = blocks

		default:
			return fmt.Errorf("message %d: unsupported content type %T", i, v)
		}
	}

	return nil
}
