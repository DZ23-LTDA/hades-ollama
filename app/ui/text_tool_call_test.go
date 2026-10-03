package ui

import "testing"

func TestParseTextToolCall(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		wantName string
		wantOK   bool
		wantQ    string // expected "query" argument, when relevant
	}{
		{"plain web_search", `{"name": "web_search", "arguments": {"query": "teste", "max_results": 5}}`, "web_search", true, "teste"},
		{"fenced json", "```json\n{\"name\": \"web_search\", \"arguments\": {\"query\": \"x\"}}\n```", "web_search", true, "x"},
		{"parameters key", `{"name":"web_fetch","parameters":{"url":"https://a"}}`, "web_fetch", true, ""},
		{"prose then braces", "Claro! Aqui vai: {}", "", false, ""},
		{"embedded, not whole", `A resposta é {"name":"web_search","arguments":{}} talvez`, "", false, ""},
		{"empty", "", "", false, ""},
		{"missing name", `{"arguments":{"query":"x"}}`, "", false, ""},
		{"not json", "web_search(teste)", "", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, args, ok := parseTextToolCall(tc.content)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if name != tc.wantName {
				t.Fatalf("name=%q want %q", name, tc.wantName)
			}
			if tc.wantQ != "" {
				if q, _ := args["query"].(string); q != tc.wantQ {
					t.Fatalf("query=%q want %q", q, tc.wantQ)
				}
			}
		})
	}
}
