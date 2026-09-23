package provider

import "testing"

// TestChatDeltaThinkingRealShapes feeds chatDeltaThinking the exact chunk JSON
// shapes real gateways send (captured live). The hook must descend into
// choices[0].delta — a top-level "delta" lookup silently matches nothing, which
// is how reasoning disappeared from the GUI with no error anywhere.
func TestChatDeltaThinkingRealShapes(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "OpenCodeZen/OpenRouter delta.reasoning",
			raw:  `{"id":"gen-1","object":"chat.completion.chunk","created":1790146827,"model":"mimo-v2.5","choices":[{"index":0,"finish_reason":null,"delta":{"role":"assistant","content":"","reasoning":"用户","reasoning_details":[{"type":"reasoning.text","text":"用户","format":"unknown","index":0}]}}]}`,
			want: "用户",
		},
		{
			name: "delta.reasoning without details",
			raw:  `{"choices":[{"delta":{"reasoning":"abc"}}]}`,
			want: "abc",
		},
		{
			name: "DeepSeek-style reasoning_content",
			raw:  `{"choices":[{"delta":{"reasoning_content":"thinking..."}}]}`,
			want: "thinking...",
		},
		{
			name: "reasoning_details underscore type",
			raw:  `{"choices":[{"delta":{"reasoning_details":[{"type":"reasoning_text","text":"detail"}]}}]}`,
			want: "detail",
		},
		{
			name: "usage chunk without choices",
			raw:  `{"object":"chat.completion.chunk","usage":{"prompt_tokens":1,"completion_tokens":2}}`,
			want: "",
		},
		{
			name: "content-only delta",
			raw:  `{"choices":[{"delta":{"content":"answer"}}]}`,
			want: "",
		},
		{
			name: "role-only first delta",
			raw:  `{"choices":[{"delta":{"role":"assistant"}}]}`,
			want: "",
		},
	}
	for _, tc := range cases {
		if got := chatDeltaThinking(tc.raw); got != tc.want {
			t.Errorf("%s: chatDeltaThinking() = %q, want %q", tc.name, got, tc.want)
		}
	}
}
