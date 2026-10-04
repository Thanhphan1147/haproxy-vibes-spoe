package main

type promptData struct {
	Method   string
	Pathq    string
	Ver      string
	Hostname string
	Headers  string
	Body     string
	Backends string
}

type ResponseMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ResponseMessageWrapper struct {
	Message ResponseMessage `json:"message"`
}

type Response struct {
	Choices []ResponseMessageWrapper `json:"choices"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Strict      bool           `json:"strict,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolChoiceFunction struct {
	Name string `json:"name"`
}

type Thinking struct {
	Type string `json:"type"`
}

type ChatRequest struct {
	Model       string            `json:"model"`
	Messages    []ResponseMessage `json:"messages"`
	Tools       []Tool            `json:"tools,omitempty"`
	ToolChoice  string            `json:"tool_choice,omitempty"`
	Thinking    *Thinking         `json:"thinking,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
}

type Decision struct {
	Allowed bool   `json:"allowed"`
	Backend string `json:"backend"`
	Reason  string `json:"reason"`
}
