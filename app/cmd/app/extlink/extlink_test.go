package extlink

import "testing"

func TestAllowedAcceptsWebAndAppSchemes(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"http://localhost:1234/", "http://localhost:1234/"},
		{"https://ollama.com/", "https://ollama.com/"},
		{"HTTPS://Ollama.com/Connect?state=x", "HTTPS://Ollama.com/Connect?state=x"},
		{"  https://ollama.com/  ", "https://ollama.com/"},
		{"ollama://connect", "ollama://connect"},
		{"OLLAMA://apps", "OLLAMA://apps"},
	}
	for _, c := range cases {
		got, ok := Allowed(c.in)
		if !ok {
			t.Errorf("Allowed(%q) = _, false; want allowed", c.in)
			continue
		}
		if got != c.want {
			t.Errorf("Allowed(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestAllowedRefusesDangerousOrUnknownSchemes(t *testing.T) {
	refused := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
		"vbscript:msgbox(1)",
		"ftp://example.com/",
		"httpx://example.com/",
		"https:/missing-slash",
		"",
		"   ",
		"not a url",
		"//evil.com",
		"https://example.com/\x00evil",
		"https://example.com/\nSet-Cookie: x",
		"https://example.com/\r\nSet-Cookie: x",
		"https://example.com/\tfoo",
	}
	for _, in := range refused {
		if got, ok := Allowed(in); ok {
			t.Errorf("Allowed(%q) = %q, true; want refused", in, got)
		}
	}
}
