package jsonrpc

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Handler func(params json.RawMessage) (interface{}, error)

type Server struct {
	handlers map[string]Handler
}

func NewServer() *Server {
	return &Server{
		handlers: make(map[string]Handler),
	}
}

func (s *Server) RegisterMethod(method string, handler Handler) {
	s.handlers[method] = handler
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, nil, -32700, "Parse error", err.Error())
		return
	}
	
	if req.JSONRPC != "2.0" {
		s.writeError(w, req.ID, -32600, "Invalid Request", "JSON-RPC version must be 2.0")
		return
	}
	
	handler, exists := s.handlers[req.Method]
	if !exists {
		s.writeError(w, req.ID, -32601, "Method not found", fmt.Sprintf("Method '%s' not found", req.Method))
		return
	}
	
	result, err := handler(req.Params)
	if err != nil {
		log.Printf("Error in method %s: %v", req.Method, err)
		s.writeError(w, req.ID, -32000, "Server error", err.Error())
		return
	}
	
	s.writeResult(w, req.ID, result)
}

func (s *Server) writeError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) writeResult(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}