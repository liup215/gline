package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestWebFetchPrivateHosts(t *testing.T) {
	tool := NewWebFetchTool()

	testCases := []struct {
		name    string
		url     string
		wantErr string
	}{
		{
			name:    "localhost",
			url:     `{"url": "http://localhost:8080/secret"}`,
			wantErr: "private",
		},
		{
			name:    "127.0.0.1",
			url:     `{"url": "http://127.0.0.1/admin"}`,
			wantErr: "private",
		},
		{
			name:    "192.168.x.x",
			url:     `{"url": "http://192.168.1.100/login"}`,
			wantErr: "private",
		},
		{
			name:    "10.x.x.x",
			url:     `{"url": "http://10.0.0.5/"}`,
			wantErr: "private",
		},
		{
			name:    "file protocol",
			url:     `{"url": "file:///etc/passwd"}`,
			wantErr: "only http and https",
		},
		{
			name:    "invalid URL",
			url:     `{"url": "://bad-url"}`,
			wantErr: "invalid URL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := tool.Execute(context.Background(), []byte(tc.url))
			if err == nil {
				// Some cases return success but the output contains "Error".
				if !strings.Contains(output, "Error:") {
					t.Fatalf("expected error or error message for %s, got none. output:\n%s", tc.url, output)
				}
			} else {
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
				}
			}
		})
	}
}

func TestWebFetchInputDecoding(t *testing.T) {
	good := `{"url": "https://example.com"}`
	var input WebFetchInput
	if err := json.Unmarshal([]byte(good), &input); err != nil {
		t.Fatalf("valid input should decode: %v", err)
	}
	if input.URL != "https://example.com" {
		t.Errorf("URL mismatch: got %q", input.URL)
	}

	bad := `{"url": true}`
	var input2 WebFetchInput
	if err := json.Unmarshal([]byte(bad), &input2); err == nil {
		t.Error("invalid input type should fail to decode")
	}
}

func TestIsPrivateHost(t *testing.T) {
	tests := []struct {
		host    string
		private bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"192.168.1.100", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.32.0.1", false},
		{"8.8.8.8", false},
		{"example.com", false},
		{"github.com", false},
	}
	for _, tc := range tests {
		got := isPrivateHost(tc.host)
		if got != tc.private {
			t.Errorf("isPrivateHost(%q) = %v, want %v", tc.host, got, tc.private)
		}
	}
}
