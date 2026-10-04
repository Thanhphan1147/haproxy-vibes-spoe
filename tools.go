package main

const decisionToolName = "submit_decision"

func buildDecisionParameters(backends []string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"allowed": map[string]any{
				"type":        "boolean",
				"description": "True if the HTTP request is safe to forward, false if it must be blocked.",
			},
			"backend": map[string]any{
				"type":        "string",
				"description": "The backend to route the request to when allowed. Must be one of the provided backends.",
				"enum":        backends,
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Short, witty explanation of the decision, addressed directly to the requester.",
			},
		},
		"required":             []string{"allowed", "backend", "reason"},
		"additionalProperties": false,
	}
}

func buildDecisionTool(backends []string) []Tool {
	tools := []Tool{{
		Type: "function",
		Function: ToolFunction{
			Name:        decisionToolName,
			Description: "Submit the analysis of the incoming HTTP request.",
			Strict:      true,
			Parameters:  buildDecisionParameters(backends),
		},
	}}
	return tools
}
