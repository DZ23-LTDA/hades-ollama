package attachments

import (
	"path"
	"strings"
)

// Noisy reports files that add size but little value when a project is
// attached to a chat: dependency lockfiles, vendored or generated output and
// minified bundles.
func Noisy(name string) bool {
	clean := strings.ToLower(strings.ReplaceAll(name, `\`, "/"))
	base := path.Base(clean)
	switch base {
	case "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb", "composer.lock",
		"cargo.lock", "poetry.lock", "gemfile.lock", "go.sum", "npm-shrinkwrap.json":
		return true
	}
	if strings.HasSuffix(base, ".min.js") || strings.HasSuffix(base, ".min.css") || strings.HasSuffix(base, ".map") {
		return true
	}
	for _, dir := range []string{"node_modules/", "dist/", "build/", ".next/", "out/", "coverage/", "vendor/", ".git/", "__pycache__/", ".venv/", "venv/"} {
		if strings.HasPrefix(clean, dir) || strings.Contains(clean, "/"+dir) {
			return true
		}
	}
	return false
}
