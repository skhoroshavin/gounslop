package ruletest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"gopkg.in/yaml.v3"
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
	FixedFiles     map[string]string
}

// Result contains the normalized command result for a single scenario run.
type Result struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Output     string
	Error      string
	FixedFiles map[string]string
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

	return runCustomGCL(t, workspace, scenario.Linter)
}

// ExecuteFix materializes the scenario into a temp workspace, runs custom-gcl --fix,
// and returns the result with fixed file contents.
func ExecuteFix(t *testing.T, scenario Scenario) Result {
	t.Helper()

	workspace := newWorkspace(t)
	writeWorkspace(t, workspace, scenario)

	result := runCustomGCLFix(t, workspace, scenario.Linter)
	result.FixedFiles = readFixedFiles(t, workspace, scenario)

	return result
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

	for path, expected := range expect.FixedFiles {
		actual, ok := result.FixedFiles[path]
		if !ok {
			t.Fatalf("expected fixed file %q but it was not written", path)
		}
		if actual != expected {
			t.Fatalf("fixed file %q mismatch:\nexpected:\n%s\nactual:\n%s", path, expected, actual)
		}
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

func runCustomGCL(t *testing.T, workspace string, _ string) Result {
	t.Helper()

	release := acquireCustomGCLLock(t)
	defer release()

	binaryPath := filepath.Join(repoRoot(), "custom-gcl")
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("custom-gcl binary is required at %s; run `make custom-gcl` first", binaryPath)
	}

	cmd := exec.Command(binaryPath, "run", "--default=none", "./...")
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

func runCustomGCLFix(t *testing.T, workspace string, _ string) Result {
	t.Helper()

	release := acquireCustomGCLLock(t)
	defer release()

	binaryPath := filepath.Join(repoRoot(), "custom-gcl")
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("custom-gcl binary is required at %s; run `make custom-gcl` first", binaryPath)
	}

	cmd := exec.Command(binaryPath, "run", "--default=none", "--fix", "./...")
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

	t.Fatalf("running custom-gcl --fix: %v", err)
	return Result{}
}

func readFixedFiles(t *testing.T, workspace string, scenario Scenario) map[string]string {
	t.Helper()

	fixed := make(map[string]string)
	for path := range scenario.Files {
		if path == "go.mod" || path == ".golangci.yml" {
			continue
		}

		fullPath := filepath.Join(workspace, path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("reading fixed file %s: %v", path, err)
		}
		fixed[path] = string(data)
	}

	return fixed
}

func acquireCustomGCLLock(t *testing.T) func() {
	t.Helper()

	lockPath := filepath.Join(os.TempDir(), "gounslop-custom-gcl.lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("opening custom-gcl lock %s: %v", lockPath, err)
	}

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		_ = lockFile.Close()
		t.Fatalf("locking custom-gcl lock %s: %v", lockPath, err)
	}

	return func() {
		if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN); err != nil {
			t.Errorf("unlocking custom-gcl lock %s: %v", lockPath, err)
		}
		if err := lockFile.Close(); err != nil {
			t.Errorf("closing custom-gcl lock %s: %v", lockPath, err)
		}
	}
}

func writeWorkspace(t *testing.T, workspace string, scenario Scenario) {
	t.Helper()

	if scenario.Linter == "" {
		t.Fatal("scenario linter is required")
	}

	if len(scenario.Files) == 0 {
		t.Fatal("scenario files are required")
	}

	files := make(map[string]string, len(scenario.Files))
	for k, v := range scenario.Files {
		files[k] = v
	}
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
	type customLinter struct {
		Type     string         `yaml:"type"`
		Settings map[string]any `yaml:"settings,omitempty"`
	}

	type lintersSettings struct {
		Custom map[string]customLinter `yaml:"custom"`
	}

	type linters struct {
		Enable   []string        `yaml:"enable"`
		Settings lintersSettings `yaml:"settings"`
	}

	type config struct {
		Version string  `yaml:"version"`
		Linters linters `yaml:"linters"`
	}

	cfg := config{
		Version: "2",
		Linters: linters{
			Enable: []string{scenario.Linter},
			Settings: lintersSettings{
				Custom: map[string]customLinter{
					scenario.Linter: {
						Type:     "module",
						Settings: scenario.Settings,
					},
				},
			},
		},
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		panic(fmt.Sprintf("rendering config: %v", err))
	}
	return string(out)
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
