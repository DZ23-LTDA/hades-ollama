// Package extlink decides which URLs the desktop app may hand to the system
// browser via the webview's openExternal binding. It lives in its own package
// (no cgo, no build tag) so the allow-list can be unit-tested on every
// platform — including the Linux CI, where the cgo-only app/cmd/app package
// itself does not build.
package extlink

import "strings"

// Allowed reports whether target is a URL openExternal may open in the system
// browser, returning the trimmed URL when so. Only http(s) and the app's own
// ollama:// scheme are allowed; javascript:, data:, file: and every other
// scheme are refused so a hostile page cannot abuse the binding.
func Allowed(target string) (string, bool) {
	target = strings.TrimSpace(target)
	// Reject any URL carrying ASCII control characters (NUL, CR, LF, TAB, …):
	// they have no legitimate place in a web URL and could be used to smuggle
	// argument boundaries or confuse the OS URL handler.
	if target == "" || hasControlChar(target) {
		return "", false
	}
	switch {
	case hasSchemePrefix(target, "http://"),
		hasSchemePrefix(target, "https://"),
		hasSchemePrefix(target, "ollama://"):
		return target, true
	default:
		return "", false
	}
}

// hasControlChar reports whether s contains any ASCII control character
// (code points below 0x20, or DEL 0x7f).
func hasControlChar(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

// hasSchemePrefix reports whether target begins with scheme, case-insensitively,
// without allocating a lower-cased copy of the whole URL.
func hasSchemePrefix(target, scheme string) bool {
	return len(target) >= len(scheme) && strings.EqualFold(target[:len(scheme)], scheme)
}
