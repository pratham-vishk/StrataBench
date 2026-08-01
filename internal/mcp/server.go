package mcp

import (
	"context"
	"encoding/json"
)

const protocolVersion = "2024-11-05"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type server struct {
	tools *Tools
}

func newServer(tools *Tools) *server {
	return &server{tools: tools}
}

func (s *server) handle(ctx context.Context, req rpcRequest) *rpcResponse {
	if req.ID == nil {
		_ = s.handleNotification(req)
		return nil
	}
	resp := &rpcResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]string{
				"name":    "stratabench-mcp",
				"version": "0.8.0",
			},
		}
	case "tools/list":
		resp.Result = map[string]any{"tools": ToolCatalog()}
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &rpcError{Code: -32602, Message: err.Error()}
			return resp
		}
		if params.Arguments == nil {
			params.Arguments = map[string]any{}
		}
		out, err := s.tools.Call(ctx, params.Name, params.Arguments)
		if err != nil {
			resp.Result = map[string]any{
				"content": []map[string]string{{"type": "text", "text": err.Error()}},
				"isError": true,
			}
			return resp
		}
		raw, _ := json.MarshalIndent(out, "", "  ")
		resp.Result = map[string]any{
			"content": []map[string]string{{"type": "text", "text": string(raw)}},
		}
	case "ping":
		resp.Result = map[string]any{}
	default:
		// notifications and unhandled methods
		if req.Method == "notifications/initialized" {
			return nil
		}
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp
}

func (s *server) handleNotification(req rpcRequest) error {
	return nil
}
