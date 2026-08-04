package main

// Core structs and the config

// json structs

type Req struct {
	Model      string    `json:"model"`
	Max_tokens int       `json:"max_tokens"`
	System     string    `json:"system"`
	Messages   []Message `json:"messages"`
	Tools      []Tool    `json:"tools"`
}

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // content block goes here, we will inject one of the two next vars
}

type Tool struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Input_schema InputSchema `json:"input_schema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// req.message.content:
type ContentBlock struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	Content   any            `json:"content,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
}

type Config struct {
	PORT string

	userAPIkey string

	stubMessage int // after how many turns we can stub the tool log message

	truncateHead int // determines the first n lines that will remain after truncation

	truncateTail int // determines the last n lines that will remain after truncation

}
