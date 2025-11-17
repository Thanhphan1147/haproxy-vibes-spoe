package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/negasus/haproxy-spoe-go/action"
	"github.com/negasus/haproxy-spoe-go/agent"
	"github.com/negasus/haproxy-spoe-go/logger"
	"github.com/negasus/haproxy-spoe-go/request"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

type Reply struct {
	Ok      bool   `json:"ok"`
	Reason  string `json:"reason,omitempty"`
	Backend string `json:"backend,omitempty"`
}

func main() {
	if len(os.Args) < 3 {
		log.Printf("Usage: %s <bind-port> <openai_api_key>\n", os.Args[0])
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

	ver, hostname, pathq, hrds, body, backendsArg, err := parseRequest(req)
	if err != nil {
		log.Printf("error parsing request: %v", err)
		return
	}

	openai_api_key := os.Args[2]
	client := openai.NewClient(
		option.WithAPIKey(openai_api_key),
		option.WithBaseURL("https://api.groq.com/openai/v1"),
	)
	template := `
You are an expert network engineer and cybersecurity analyst,
you can identify malicious requests simply by looking at them.
You can also know where to route requests just by looking at them.

Given the following HTTP request, determine if it is malicious or safe.
Respond in valid json format with this structure (THE RESPONSE MUST BE PARSABLE WITH ANY JSON PARSER, DON'T ADD BACKTICKS OR MARKDOWN FORMATTING. DO NOT USE M-DASHES):
{
	"ok": True, # If the request is safe, set this to True, else False if the request is malicious.
	"reason": "Only add this field if ok != True. Imagine you are a world-class detective dramatically uncovering the villain's scheme in real time, greeting your archenemy when stepping out of the shadow, giving him the brutal truth to his face before taking him to jail. Give the unfortunate hacker a nickname, bond-villain style. Add a touch of wit, drama and humor. Talk in first person, towards the villain"
	"backend": Only add this field if ok == True. Choose the backend that will get this request.
}

Main information about the request:
HTTP %s
%s%s
%s
%s

The backends available are:
%s
`
	input := fmt.Sprintf(template, ver, hostname, pathq, hrds, body, backendsArg)
	fmt.Printf("input: %s\n", input)

	resp, err := client.Responses.New(context.TODO(), responses.ResponseNewParams{
		Model: "openai/gpt-oss-20b",
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(input),
		},
	})
	if err != nil {
		panic(err.Error())
	}

	var reply Reply
	err = json.Unmarshal([]byte(resp.OutputText()), &reply)
	if err != nil {
		log.Printf("error parsing response: %v", err)
		return
	}
	fmt.Printf("response: %+v\n", resp.OutputText())
	req.Actions.SetVar(action.ScopeSession, "ok", reply.Ok)
	req.Actions.SetVar(action.ScopeSession, "reason", reply.Reason)
	req.Actions.SetVar(action.ScopeSession, "backend", reply.Backend)
}
