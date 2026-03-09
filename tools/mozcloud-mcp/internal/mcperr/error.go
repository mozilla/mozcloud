// Package mcperr provides a shared error type for all MCP tool responses.
// Every error returned by a tool MUST use this type and MUST have a non-empty
// Remediation field telling the user exactly what to do.
package mcperr

import "encoding/json"

// MCPError is the structured error payload embedded in every error response.
type MCPError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

// ErrorResponse is the top-level JSON object returned on failure.
type ErrorResponse struct {
	Error MCPError `json:"error"`
}

// New constructs an ErrorResponse. remediation must not be empty.
func New(code, message, remediation string) ErrorResponse {
	return ErrorResponse{
		Error: MCPError{
			Code:        code,
			Message:     message,
			Remediation: remediation,
		},
	}
}

// JSON serialises the response to a JSON string. If marshalling fails
// (which should never happen for this simple struct) it returns a
// hardcoded fallback so callers always get a valid string.
func (e ErrorResponse) JSON() string {
	b, err := json.Marshal(e)
	if err != nil {
		return `{"error":{"code":"marshal_error","message":"failed to marshal error","remediation":"File a bug against mozcloud-mcp"}}`
	}
	return string(b)
}
