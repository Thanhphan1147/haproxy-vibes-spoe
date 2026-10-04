package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"github.com/negasus/haproxy-spoe-go/action"
	"github.com/negasus/haproxy-spoe-go/agent"
	"github.com/negasus/haproxy-spoe-go/logger"
	"github.com/negasus/haproxy-spoe-go/request"
)

//go:embed prompt.md
var promptFS embed.FS

var promptTmpl *template.Template

func renderPrompt(name string, data promptData) (string, error) {
	var buf bytes.Buffer
	if err := promptTmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("render prompt %q: %w", name, err)
	}
	return strings.TrimSpace(buf.String()), nil
}

var (
	apiURL   string
	apiKey   string
	apiModel string
	client   = &http.Client{Timeout: 60 * time.Second}
)

func chatCompletion(messages []ResponseMessage, tools []Tool) (string, error) {
	temperature := 0.0
	payload := ChatRequest{
		Model:       apiModel,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "required",
		Thinking:    &Thinking{Type: "disabled"},
		Temperature: &temperature,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-OK status: %s, %s", resp.Status, string(respBody))
	}

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("response has no choices")
	}

	message := result.Choices[0].Message
	if len(message.ToolCalls) > 0 {
		return strings.TrimSpace(message.ToolCalls[0].Function.Arguments), nil
	}

	return strings.TrimSpace(message.Content), nil
}

func parseDecision(content string) (Decision, error) {
	var decision Decision
	if err := json.Unmarshal([]byte(content), &decision); err != nil {
		return decision, fmt.Errorf("unmarshal decision: %w", err)
	}
	return decision, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Printf("Usage: %s <bind-port>\n", os.Args[0])
		os.Exit(1)
	}
	port := os.Args[1]

	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file loaded: %v", err)
	}

	apiKey = os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Printf("OPENAI_API_KEY is not set")
		os.Exit(1)
	}

	apiURL = os.Getenv("OPENAI_API_URL")
	if apiURL == "" {
		apiURL = "https://api.deepseek.com/v1/chat/completions"
	}

	apiModel = os.Getenv("OPENAI_MODEL")
	if apiModel == "" {
		apiModel = "deepseek-flash"
	}

	tmpl, err := template.ParseFS(promptFS, "prompt.md")
	if err != nil {
		log.Printf("error parsing prompt.md: %v", err)
		os.Exit(1)
	}
	promptTmpl = tmpl

	listener, err := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%s", port))
	if err != nil {
		log.Printf("error create listener, %v", err)
		os.Exit(1)
	}
	defer listener.Close()

	a := agent.New(handler, logger.NewDefaultLog())

	if err := a.Serve(listener); err != nil {
		log.Printf("error agent serve: %+v\n", err)
	}
}

func handler(req *request.Request) {

	ver, method, hostname, pathq, hrds, body, backendsArg, err := parseRequest(req)
	if err != nil {
		log.Printf("error parsing request: %v", err)
		return
	}

	backends := make([]string, 0)
	for _, backend := range strings.Split(backendsArg, "|") {
		if backend = strings.TrimSpace(backend); backend != "" {
			backends = append(backends, backend)
		}
	}

	systemMessage, err := renderPrompt("system", promptData{})
	if err != nil {
		log.Printf("error rendering system prompt: %v", err)
		return
	}

	userMessage, err := renderPrompt("user", promptData{
		Method:   method,
		Pathq:    pathq,
		Ver:      ver,
		Hostname: hostname,
		Headers:  hrds,
		Body:     body,
		Backends: strings.Join(backends, "\n"),
	})
	if err != nil {
		log.Printf("error rendering user prompt: %v", err)
		return
	}

	tools := buildDecisionTool(backends)

	content, err := chatCompletion([]ResponseMessage{
		{Role: "system", Content: systemMessage},
		{Role: "user", Content: userMessage},
	}, tools)
	if err != nil {
		fmt.Printf("Error getting decision: %v\n", err)
		return
	}
	fmt.Printf("%s\n", content)

	decision, err := parseDecision(content)
	if err != nil {
		log.Printf("error parsing decision: %v", err)
		return
	}

	req.Actions.SetVar(action.ScopeSession, "allowed", decision.Allowed)
	req.Actions.SetVar(action.ScopeSession, "backend", strings.ToLower(decision.Backend))
	req.Actions.SetVar(action.ScopeSession, "reason", fmt.Sprintf("You request has been denied - %s", decision.Reason))
}
