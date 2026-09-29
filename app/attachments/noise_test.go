package attachments

import "testing"

func TestNoisy(t *testing.T) {
	for _, name := range []string{
		"proj.zip/package-lock.json",
		"proj.zip/node_modules/react/index.js",
		"proj.zip/apps/web/dist/main.js",
		"proj.zip/public/app.min.js",
		`proj.zip\build\out.txt`,
		"node_modules/x.js",
	} {
		if !Noisy(name) {
			t.Errorf("%s should be noisy", name)
		}
	}
	for _, name := range []string{"proj.zip/src/app.tsx", "README.md", "proj.zip/lib/distance.go", "src/distributor.ts"} {
		if Noisy(name) {
			t.Errorf("%s should be kept", name)
		}
	}
}
