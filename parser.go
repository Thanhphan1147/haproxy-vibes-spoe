package main

import (
	"fmt"
	"log"

	"github.com/negasus/haproxy-spoe-go/request"
)

func parseRequest(req *request.Request) (string, string, string, string, string, string, error) {
	log.Printf(
		"handle request EngineID: '%s', StreamID: '%d', FrameID: '%d' with %d messages\n",
		req.EngineID, req.StreamID, req.FrameID, req.Messages.Len(),
	)

	mes, _ := req.Messages.GetByIndex(0)

	ver, ok := mes.KV.Get("arg_ver")
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("var 'ver' not found in message")
	}

	hostname, ok := mes.KV.Get("arg_hostname")
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("var 'hostname' not found in message")
	}

	pathq, ok := mes.KV.Get("arg_pathq")
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("var 'pathq' not found in message")
	}

	hrds, ok := mes.KV.Get("arg_hrds")
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("var 'hrds' not found in message")
	}

	body, ok := mes.KV.Get("arg_body")
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("var 'body' not found in message")
	}

	backendsArg, ok := mes.KV.Get("arg_backends")
	if !ok {
		return "", "", "", "", "", "", fmt.Errorf("var 'backends' not found in message")
	}
	return ver.(string), hostname.(string), pathq.(string), hrds.(string), body.(string), backendsArg.(string), nil
}
