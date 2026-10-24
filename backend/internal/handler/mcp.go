package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPSettings reads the independently stored organization MCP switch.
type MCPSettings interface {
	MCPEnabled(context.Context, uuid.UUID) (bool, error)
}

// MCPHandler creates a permission-filtered stateless server for each request.
// Business handlers are invoked directly; no internal HTTP calls or sessions.
func (h *Handler) MCPHandler(operations []Operation, keys MCPSettings) http.Handler {
	transport := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		server := mcp.NewServer(&mcp.Implementation{Name: "attic", Version: "1.0.0"}, nil)
		key, user := auth.GetAPIKey(r.Context()), auth.GetUser(r.Context())
		for _, op := range operations {
			if op.ToolName == "" || op.Interactive || (op.Admin && user.Role != "admin") || !op.keyAllows(key, user) {
				continue
			}
			if op.Feature != "" {
				enabled, err := h.featureEnabled(r, op.Feature)
				if err != nil || !enabled {
					continue
				}
			}
			destructive := op.Action == "write" && op.Method != "POST"
			schema := operationSchema(op)
			encodedSchema, _ := json.Marshal(schema)
			var typedSchema jsonschema.Schema
			if err := json.Unmarshal(encodedSchema, &typedSchema); err != nil {
				panic(err)
			}
			resolved, err := typedSchema.Resolve(nil)
			if err != nil {
				panic(err)
			}
			server.AddTool(&mcp.Tool{Name: op.ToolName, Description: op.Description, InputSchema: schema, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: op.Action == "read", DestructiveHint: &destructive}}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				// SDK context carries the authenticated HTTP request context.
				if auth.GetAPIKey(ctx) == nil {
					return toolError("API key required"), nil
				}
				rawArguments := req.Params.Arguments
				if len(rawArguments) == 0 {
					rawArguments = json.RawMessage(`{}`)
				}
				var arguments any
				if err := json.Unmarshal(rawArguments, &arguments); err != nil {
					return toolError("invalid arguments"), nil
				}
				if err := resolved.Validate(arguments); err != nil {
					return toolError(err.Error()), nil
				}
				request, err := operationRequest(ctx, op, rawArguments)
				if err != nil {
					return toolError(err.Error()), nil
				}
				recorder := httptest.NewRecorder()
				h.AuthorizeOperation(op, op.Handler).ServeHTTP(recorder, request)
				result := map[string]any{"status": recorder.Code}
				if recorder.Body.Len() > 0 {
					var body any
					if json.Unmarshal(recorder.Body.Bytes(), &body) != nil {
						body = recorder.Body.String()
					}
					result["body"] = body
				}
				encoded, _ := json.Marshal(result)
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(encoded)}}, StructuredContent: result, IsError: recorder.Code >= 400}, nil
			})
		}
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 72 << 20})
	protected := http.NewCrossOriginProtection().Handler(transport)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.GetAPIKey(r.Context()) == nil {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, 401, "API key required")
			return
		}
		enabled, err := keys.MCPEnabled(r.Context(), h.orgID)
		if err != nil {
			writeError(w, 503, "MCP setting unavailable")
			return
		}
		if !enabled {
			writeError(w, 404, "MCP is disabled")
			return
		}
		protected.ServeHTTP(w, r)
	})
}
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: message}}}
}

func operationRequest(ctx context.Context, op Operation, args json.RawMessage) (*http.Request, error) {
	var input map[string]json.RawMessage
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	route := chi.NewRouteContext()
	path := op.Path
	for _, part := range strings.Split(op.Path, "/") {
		if !strings.HasPrefix(part, "{") {
			continue
		}
		name := strings.Trim(part, "{}")
		var value string
		if err := json.Unmarshal(input[name], &value); err != nil || value == "" {
			return nil, fmt.Errorf("%s is required", name)
		}
		route.URLParams.Add(name, value)
		path = strings.ReplaceAll(path, part, url.PathEscape(value))
		delete(input, name)
	}
	query := url.Values{}
	if raw, ok := input["query"]; ok {
		var values map[string]any
		if json.Unmarshal(raw, &values) != nil {
			return nil, fmt.Errorf("invalid query")
		}
		for key, value := range values {
			allowed := false
			for _, name := range op.Query {
				if key == name {
					allowed = true
				}
			}
			if !allowed {
				return nil, fmt.Errorf("unknown query parameter %s", key)
			}
			switch v := value.(type) {
			case string:
				query.Add(key, v)
			case []any:
				for _, item := range v {
					raw, ok := item.(string)
					if !ok {
						return nil, fmt.Errorf("query values must be strings")
					}
					query.Add(key, raw)
				}
			default:
				return nil, fmt.Errorf("query values must be strings")
			}
		}
		delete(input, "query")
	}
	body := input["body"]
	delete(input, "body")
	if op.Body != nil && len(body) == 0 {
		return nil, fmt.Errorf("body is required")
	}
	var impact string
	if raw, ok := input["impact_token"]; ok {
		if op.Resource != "attributes" || op.Action != "write" || json.Unmarshal(raw, &impact) != nil {
			return nil, fmt.Errorf("invalid impact_token")
		}
		delete(input, "impact_token")
	}
	if len(input) > 0 {
		return nil, fmt.Errorf("unknown arguments")
	}
	request, err := http.NewRequestWithContext(context.WithValue(ctx, chi.RouteCtxKey, route), op.Method, path+"?"+query.Encode(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if impact != "" {
		request.Header.Set("X-Impact-Token", impact)
	}
	if op.ToolName == "upload_attachment" {
		return attachmentRequest(request, body)
	}
	return request, nil
}

func attachmentRequest(r *http.Request, raw []byte) (*http.Request, error) {
	var body struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Description string `json:"description"`
		Content     string `json:"content"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return nil, err
	}
	if body.Filename == "" || body.ContentType == "" {
		return nil, fmt.Errorf("filename and content_type are required")
	}
	data, err := decodeAttachmentContent(body.Content, maxUploadSize)
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	header := textproto.MIMEHeader{}
	// FormatMediaType quotes filenames safely, including embedded quotes.
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": body.Filename}))
	header.Set("Content-Type", body.ContentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, err
	}
	if _, err = part.Write(data); err != nil {
		return nil, err
	}
	if body.Description != "" {
		if err = writer.WriteField("description", body.Description); err != nil {
			return nil, err
		}
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(&buffer)
	r.ContentLength = int64(buffer.Len())
	r.Header.Set("Content-Type", writer.FormDataContentType())
	return r, nil
}

func decodeAttachmentContent(content string, limit int) ([]byte, error) {
	if len(content) > base64.StdEncoding.EncodedLen(limit) {
		return nil, fmt.Errorf("file exceeds upload limit")
	}
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 content")
	}
	if len(data) > limit {
		return nil, fmt.Errorf("file exceeds upload limit")
	}
	return data, nil
}
