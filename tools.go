package main

const decisionToolName = "submit_decision"

func buildDecisionParameters(backends []string) map[string]any {
	reason := map[string]any{
		"type":        "string",
		"description": "Short, witty explanation of the decision, addressed directly to the requester.",
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"decision": map[string]any{
				"anyOf": []any{
					map[string]any{
						"type": "object",
						"properties": map[string]any{
							"action": map[string]any{
								"type": "string",
								"enum": []string{"route"},
							},
							"backend": map[string]any{
								"type":        "string",
								"description": "The backend to route the request to.",
								"enum":        backends,
							},
							"reason": reason,
						},
						"required":             []string{"action", "backend", "reason"},
						"additionalProperties": false,
					},
					map[string]any{
						"type": "object",
						"properties": map[string]any{
							"action": map[string]any{
								"type": "string",
								"enum": []string{"block"},
							},
							"reason": reason,
						},
						"required":             []string{"action", "reason"},
						"additionalProperties": false,
					},
					map[string]any{
						"type": "object",
						"properties": map[string]any{
							"action": map[string]any{
								"type": "string",
								"enum": []string{"respond"},
							},
							"custom_html_content": map[string]any{
								"type":        "string",
								"description": "A full, complete and correct HTML page to return to the requester.",
							},
							"reason": reason,
						},
						"required":             []string{"action", "custom_html_content", "reason"},
						"additionalProperties": false,
					},
				},
			},
		},
		"required":             []string{"decision"},
		"additionalProperties": false,
	}
}

func buildDecisionTool(backends []string) []Tool {
	return []Tool{{
		Type: "function",
		Function: ToolFunction{
			Name:        decisionToolName,
			Description: "Submit the chosen action and supporting details for the incoming HTTP request.",
			Strict:      true,
			Parameters:  buildDecisionParameters(backends),
		},
	}}
}
