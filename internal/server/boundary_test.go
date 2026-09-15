package server_test

import (
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"testing"
)

const modulePrefix = "github.com/DenisKorkmaz/nostra/internal/"

// allowedDeps names, per module under internal/, the other modules it may
// depend on. Go's internal rule hides a module's guts but says nothing about
// who may import its public surface — that second rule lives here. platform is
// infrastructure and always allowed; a new module needs a line of its own.
var allowedDeps = map[string][]string{
	"server": {"system"},
	"system": {},
}

// TestModuleDependencies asks the Go tool for the full dependency list of every
// module and fails on edges that allowedDeps does not permit. It is a build-time
// question, so it needs no database and no container.
func TestModuleDependencies(t *testing.T) {
	for _, module := range modules(t) {
		allowed, ok := allowedDeps[module]
		if !ok {
			t.Errorf("module %q has no entry in allowedDeps", module)

			continue
		}

		for _, dep := range dependencies(t, module) {
			if dep == module || dep == "platform" || slices.Contains(allowed, dep) {
				continue
			}

			t.Errorf("module %q depends on %q, which allowedDeps does not permit", module, dep)
		}
	}
}

// modules lists the directories directly under internal/, which is where the
// modules live. platform is infrastructure rather than a module.
func modules(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir("..")
	if err != nil {
		t.Fatalf("read internal/: %v", err)
	}

	var names []string

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "platform" {
			names = append(names, entry.Name())
		}
	}

	return names
}

// dependencies returns the module names that the given module reaches, directly
// or indirectly. Everything outside internal/ is irrelevant here.
func dependencies(t *testing.T, module string) []string {
	t.Helper()

	out, err := exec.Command("go", "list", "-deps", modulePrefix+module+"/...").CombinedOutput()
	if err != nil {
		t.Fatalf("go list for module %q: %v\n%s", module, err, out)
	}

	var names []string

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		pkg, found := strings.CutPrefix(strings.TrimSpace(line), modulePrefix)
		if !found {
			continue
		}

		name, _, _ := strings.Cut(path.Clean(pkg), "/")
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}

	return names
}
