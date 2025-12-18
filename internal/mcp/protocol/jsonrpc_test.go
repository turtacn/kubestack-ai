package protocol

import (
	"encoding/json"
	"testing"
)

func TestJSONRPC_EncodeRequest(t *testing.T) {
	codec := NewCodec()

	tests := []struct {
		name       string
		id         any
		method     string
		params     any
		wantMethod string
		wantID     any
	}{
		{
			name:       "simple request with int ID",
			id:         1,
			method:     "test/method",
			params:     map[string]any{"key": "value"},
			wantMethod: "test/method",
			wantID:     float64(1), // JSON numbers become float64
		},
		{
			name:       "request with string ID",
			id:         "req-123",
			method:     "another/method",
			params:     nil,
			wantMethod: "another/method",
			wantID:     "req-123",
		},
		{
			name:       "notification (nil ID)",
			id:         nil,
			method:     "notify",
			params:     map[string]string{"msg": "hello"},
			wantMethod: "notify",
			wantID:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := codec.EncodeRequest(tt.id, tt.method, tt.params)
			if err != nil {
				t.Fatalf("EncodeRequest failed: %v", err)
			}

			// Parse result
			var req map[string]any
			if err := json.Unmarshal(data, &req); err != nil {
				t.Fatalf("Failed to parse encoded request: %v", err)
			}

			if req["jsonrpc"] != "2.0" {
				t.Errorf("Expected jsonrpc=2.0, got %v", req["jsonrpc"])
			}

			if req["method"] != tt.wantMethod {
				t.Errorf("Expected method=%s, got %v", tt.wantMethod, req["method"])
			}

			if tt.wantID != nil {
				if req["id"] != tt.wantID {
					t.Errorf("Expected id=%v, got %v", tt.wantID, req["id"])
				}
			}
		})
	}
}

func TestJSONRPC_DecodeRequest(t *testing.T) {
	codec := NewCodec()

	tests := []struct {
		name      string
		input     string
		wantError bool
		wantID    any
		wantMethod string
	}{
		{
			name:       "valid request",
			input:      `{"jsonrpc":"2.0","id":1,"method":"test","params":{"x":1}}`,
			wantError:  false,
			wantID:     float64(1),
			wantMethod: "test",
		},
		{
			name:      "missing jsonrpc",
			input:     `{"id":1,"method":"test"}`,
			wantError: true,
		},
		{
			name:      "wrong jsonrpc version",
			input:     `{"jsonrpc":"1.0","id":1,"method":"test"}`,
			wantError: true,
		},
		{
			name:      "missing method",
			input:     `{"jsonrpc":"2.0","id":1}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := codec.DecodeRequest([]byte(tt.input))
			if tt.wantError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if req.Method != tt.wantMethod {
				t.Errorf("Expected method=%s, got %s", tt.wantMethod, req.Method)
			}
		})
	}
}

func TestJSONRPC_EncodeResponse(t *testing.T) {
	codec := NewCodec()

	data, err := codec.EncodeResponse(1, map[string]string{"result": "success"})
	if err != nil {
		t.Fatalf("EncodeResponse failed: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc=2.0, got %v", resp["jsonrpc"])
	}

	if resp["id"] != float64(1) {
		t.Errorf("Expected id=1, got %v", resp["id"])
	}

	if resp["result"] == nil {
		t.Error("Expected result field")
	}
}

func TestJSONRPC_EncodeError(t *testing.T) {
	codec := NewCodec()

	data, err := codec.EncodeError(1, MethodNotFound, "Method not found", "details")
	if err != nil {
		t.Fatalf("EncodeError failed: %v", err)
	}

	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc=2.0, got %s", resp.JSONRPC)
	}

	if resp.Error == nil {
		t.Fatal("Expected error field")
	}

	if resp.Error.Code != MethodNotFound {
		t.Errorf("Expected code=%d, got %d", MethodNotFound, resp.Error.Code)
	}
}

func TestJSONRPC_DecodeResponse(t *testing.T) {
	codec := NewCodec()

	tests := []struct {
		name      string
		input     string
		wantError bool
		hasError  bool
	}{
		{
			name:      "successful response",
			input:     `{"jsonrpc":"2.0","id":1,"result":{"data":"test"}}`,
			wantError: false,
			hasError:  false,
		},
		{
			name:      "error response",
			input:     `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Not found"}}`,
			wantError: false,
			hasError:  true,
		},
		{
			name:      "invalid json",
			input:     `{invalid}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := codec.DecodeResponse([]byte(tt.input))
			if tt.wantError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.hasError {
				if resp.Error == nil {
					t.Error("Expected error field in response")
				}
			} else {
				if resp.Result == nil {
					t.Error("Expected result field in response")
				}
			}
		})
	}
}

func TestRequest_IsNotification(t *testing.T) {
	tests := []struct {
		name string
		req  Request
		want bool
	}{
		{
			name: "notification with nil ID",
			req:  Request{ID: nil, Method: "notify"},
			want: true,
		},
		{
			name: "request with int ID",
			req:  Request{ID: 1, Method: "call"},
			want: false,
		},
		{
			name: "request with string ID",
			req:  Request{ID: "123", Method: "call"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.IsNotification(); got != tt.want {
				t.Errorf("IsNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}
