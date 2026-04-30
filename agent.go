package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/negasus/haproxy-spoe-go/action"
	"github.com/negasus/haproxy-spoe-go/agent"
	"github.com/negasus/haproxy-spoe-go/logger"
	"github.com/negasus/haproxy-spoe-go/request"

	"net/http"
)

type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type ResponseMessageWrapper struct {
	Message ResponseMessage `json:"message"`
}
type Response struct {
	Choices []ResponseMessageWrapper `json:"choices"`
}

type ChatRequest struct {
	Messages []ResponseMessage `json:"messages"`
}

func chatCompletion(client *http.Client, url string, messages []ResponseMessage) (string, error) {
	payload := ChatRequest{Messages: messages}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

func main() {
	if len(os.Args) < 2 {
		log.Printf("Usage: %s <bind-port>\n", os.Args[0])
		os.Exit(1)
	}
	port := os.Args[1]

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

	systemMessage := "You are an expert network engineer and cybersecurity analyst,you can identify malicious requests simply by looking at them. You can also know where to route requests just by looking at them.\nThis is an HTTP request coming into an HAProxy loadbalancer and you've been tasked to analyze it and decide if the request is safe or not. Your response will be used by HAProxy to decide what to do with the incoming HTTP request. If the request has been determined to be safe by you, it will be forwarded to the servers behind the load balancer. There are 3 types of questions you will be asked:\n1. 'Is this HTTP request safe?' - Reply with ONLY 'Yes' or 'No', DON'T ADD ANYTHING ELSE. You tend to say 'Yes' if you don't see anything particularly wrong with the incoming HTTP request\n2. 'Which backend to route this request to? Pick one from the list' - When asked this question the user will give you a list of HAProxy backends to choose from, for example:default\nweb\napi, in this example you can reply with either 'default', 'web' or 'api' depending on your analysis of the HTTP request.\n3. Why did you block this HTTP request? - Give a short witty explaination on why this HTTP request is dangerous and should be blocked."

	user_message := fmt.Sprintf("Is this HTTP request safe? Here's the information about the incoming HTTP request:\n%s %s HTTP/%s\nHost: %s\nHeaders: %s\nBody: %s\n /no_think", method, pathq, ver, hostname, hrds, body)
	apiURL := "http://127.0.0.1:11434/chat/completions"
	client := &http.Client{}
	content, err := chatCompletion(client, apiURL, []ResponseMessage{
		{Role: "system", Content: systemMessage},
		{Role: "user", Content: user_message},
	})
	if err != nil {
		fmt.Printf("Error getting allow decision: %v\n", err)
		return
	}
	fmt.Printf("%s\n", content)

	allowed := strings.Contains(strings.ToLower(content), "yes")
	backends := strings.Split(backendsArg, "|")
	var availableBackends strings.Builder
	for i := 0; i < len(backends); i++ {
		availableBackends.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(backends[i])))
	}

	reason := ""
	if !allowed {
		reason, err = chatCompletion(client, apiURL, []ResponseMessage{
			{Role: "system", Content: systemMessage},
			{Role: "user", Content: user_message},
			{Role: "assistant", Content: content},
			{Role: "user", Content: "Why? - Give a short witty explaination to elaborate on your previous answer. Imagine there's an attacker sending the request and you are talking to them, exposing their plans in real time. /no_think"},
		})
		if err != nil {
			fmt.Printf("Error getting backend decision: %v\n", err)
			return
		}
		fmt.Printf("Reason: %s\n", reason)
	}

	target := "default"
	if allowed {
		target, err = chatCompletion(client, apiURL, []ResponseMessage{
			{Role: "system", Content: systemMessage},
			{Role: "user", Content: user_message},
			{Role: "assistant", Content: content},
			{Role: "user", Content: fmt.Sprintf("Which backend amongst these? Pick one from the list:\n%s /no_think", availableBackends.String())},
		})
		if err != nil {
			fmt.Printf("Error getting backend decision: %v\n", err)
			return
		}
		fmt.Printf("%s\n", target)
	}
	req.Actions.SetVar(action.ScopeSession, "allowed", allowed)
	req.Actions.SetVar(action.ScopeSession, "backend", strings.ToLower(target))
	req.Actions.SetVar(action.ScopeSession, "reason", fmt.Sprintf("You request has been denied - %s", reason))
}
