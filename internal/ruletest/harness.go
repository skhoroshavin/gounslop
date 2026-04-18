package ruletest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// Scenario defines a temporary workspace, generated linter config, and expected outcome.
type Scenario struct {
	Name       string
	ModulePath string
	GoVersion  string
	Linter     string
	Files      map[string]string
	Settings   map[string]any
	Expect     Expectation
}

// Expectation defines the stable fragments that must appear in normalized output.
type Expectation struct {
	ExitCode       int
	OutputContains []string
	EmptyOutput    bool
}

// Result contains the normalized command result for a single scenario run.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Output   string
	Error    string
}

// Run materializes the scenario into a temp workspace, executes custom-gcl, and checks expectations.
func Run(t *testing.T, scenario Scenario) Result {
	t.Helper()

	result := Execute(t, scenario)
	AssertResult(t, scenario.Expect, result)

	return result
}

// Execute materializes the scenario into a temp workspace, runs custom-gcl, and returns the normalized result.
func Execute(t *testing.T, scenario Scenario) Result {
	t.Helper()

	workspace := newWorkspace(t)
	writeWorkspace(t, workspace, scenario)

	return runCustomGCL(t, workspace)
}

// AssertResult validates a normalized E2E command result against the expected outcome.
func AssertResult(t *testing.T, expect Expectation, result Result) {
	t.Helper()

	if result.ExitCode != expect.ExitCode {
		t.Fatalf(
			"unexpected exit code: got %d, want %d\nerror:\n%s\nstdout:\n%s\nstderr:\n%s",
			result.ExitCode,
			expect.ExitCode,
			result.Error,
			result.Stdout,
			result.Stderr,
		)
	}

	if expect.EmptyOutput && !isEmptyOutput(result.Output) {
		t.Fatalf("expected no output, got:\n%s", result.Output)
	}

	for _, fragment := range expect.OutputContains {
		if strings.Contains(result.Output, fragment) {
			continue
		}

		t.Fatalf("missing output fragment %q\nfull output:\n%s", fragment, result.Output)
	}
}

func newWorkspace(t *testing.T) string {
	t.Helper()

	workspace, err := os.MkdirTemp("", "gounslop-plugin-e2e-*")
	if err != nil {
		t.Fatalf("creating temp workspace: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(workspace); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("removing temp workspace %s: %v", workspace, err)
		}
	})

	return workspace
}

func runCustomGCL(t *testing.T, workspace string) Result {
	t.Helper()

	binaryPath := filepath.Join(repoRoot(), "custom-gcl")
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("custom-gcl binary is required at %s; run `make custom-gcl` first", binaryPath)
	}

	cmd := exec.Command(binaryPath, "run", "./...")
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		ExitCode: exitCode(err),
		Stdout:   normalizeOutput(stdout.String(), workspace),
		Stderr:   normalizeOutput(stderr.String(), workspace),
	}
	if err != nil {
		result.Error = err.Error()
	}
	result.Output = strings.TrimSpace(strings.Join(nonEmpty(result.Stdout, result.Stderr), "\n"))

	if err == nil {
		return result
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return result
	}

	t.Fatalf("running custom-gcl: %v", err)
	return Result{}
}

func writeWorkspace(t *testing.T, workspace string, scenario Scenario) {
	t.Helper()

	if scenario.Linter == "" {
		t.Fatal("scenario linter is required")
	}

	if len(scenario.Files) == 0 {
		t.Fatal("scenario files are required")
	}

	files := mapsClone(scenario.Files)
	if _, ok := files["go.mod"]; !ok {
		files["go.mod"] = renderGoMod(scenario)
	}
	if _, ok := files[".golangci.yml"]; !ok {
		files[".golangci.yml"] = renderConfig(scenario)
	}

	for path, content := range files {
		fullPath := filepath.Join(workspace, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("creating %s parent directories: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
}

func renderConfig(scenario Scenario) string {
	var builder strings.Builder

	builder.WriteString("version: \"2\"\n\n")
	builder.WriteString("linters:\n")
	builder.WriteString("  enable:\n")
	_, _ = fmt.Fprintf(&builder, "    - %s\n", scenario.Linter)
	builder.WriteString("  settings:\n")
	builder.WriteString("    custom:\n")
	_, _ = fmt.Fprintf(&builder, "      %s:\n", scenario.Linter)
	builder.WriteString("        type: \"module\"\n")

	if len(scenario.Settings) == 0 {
		return builder.String()
	}

	builder.WriteString("        settings:\n")
	writeYAMLMap(&builder, 10, scenario.Settings)

	return builder.String()
}

func renderGoMod(scenario Scenario) string {
	modulePath := scenario.ModulePath
	if modulePath == "" {
		modulePath = "example.com/plugin-e2e"
	}

	goVersion := scenario.GoVersion
	if goVersion == "" {
		goVersion = "1.25.6"
	}

	return fmt.Sprintf("module %s\n\ngo %s\n", modulePath, goVersion)
}

func writeYAMLMap(builder *strings.Builder, indent int, values map[string]any) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		writeYAMLValue(builder, indent, key, values[key])
	}
}

func writeYAMLList(builder *strings.Builder, indent int, values []any) {
	padding := strings.Repeat(" ", indent)

	for _, value := range values {
		switch item := value.(type) {
		case map[string]any:
			builder.WriteString(padding + "-\n")
			writeYAMLMap(builder, indent+2, item)
		case []any:
			builder.WriteString(padding + "-\n")
			writeYAMLList(builder, indent+2, item)
		default:
			_, _ = fmt.Fprintf(builder, "%s- %s\n", padding, yamlScalar(item))
		}
	}
}

func writeYAMLValue(builder *strings.Builder, indent int, key string, value any) {
	padding := strings.Repeat(" ", indent)

	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			_, _ = fmt.Fprintf(builder, "%s%s: {}\n", padding, key)
			return
		}

		_, _ = fmt.Fprintf(builder, "%s%s:\n", padding, key)
		writeYAMLMap(builder, indent+2, typed)
	case []any:
		if len(typed) == 0 {
			_, _ = fmt.Fprintf(builder, "%s%s: []\n", padding, key)
			return
		}

		_, _ = fmt.Fprintf(builder, "%s%s:\n", padding, key)
		writeYAMLList(builder, indent+2, typed)
	case []string:
		if len(typed) == 0 {
			_, _ = fmt.Fprintf(builder, "%s%s: []\n", padding, key)
			return
		}

		_, _ = fmt.Fprintf(builder, "%s%s:\n", padding, key)
		items := make([]any, len(typed))
		for i, item := range typed {
			items[i] = item
		}
		writeYAMLList(builder, indent+2, items)
	default:
		_, _ = fmt.Fprintf(builder, "%s%s: %s\n", padding, key, yamlScalar(value))
	}
}

func yamlScalar(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return fmt.Sprintf("%q", typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case fmt.Stringer:
		return fmt.Sprintf("%q", typed.String())
	default:
		return fmt.Sprint(value)
	}
}

func normalizeOutput(output string, workspace string) string {
	trimmed := strings.ReplaceAll(output, "\r\n", "\n")
	trimmed = strings.ReplaceAll(trimmed, workspace, "<workspace>")
	trimmed = strings.ReplaceAll(trimmed, filepath.ToSlash(workspace), "<workspace>")
	return strings.TrimSpace(trimmed)
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to determine repository root")
	}

	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return -1
}

func mapsClone(values map[string]string) map[string]string {
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		result = append(result, value)
	}
	return result
}

func isEmptyOutput(output string) bool {
	trimmed := strings.TrimSpace(output)
	return trimmed == "" || trimmed == "0 issues."
}
