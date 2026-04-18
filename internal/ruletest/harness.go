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

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type Suite struct {
	suite.Suite

	Linter     string
	ModulePath string
	GoVersion  string

	files          map[string]string
	settings       map[string]any
	workspace      string
	lastResult     *Result
	lastTargetPath string
	codeCounter    int
}

type Result struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Output     string
	Error      string
	FixedFiles map[string]string
}

func (s *Suite) SetupTest() {
	s.files = make(map[string]string)
	s.settings = nil
	s.workspace = ""
	s.lastResult = nil
	s.lastTargetPath = ""
	s.codeCounter = 0
}

func (s *Suite) GivenConfig(settings map[string]any) {
	s.T().Helper()

	if settings == nil {
		s.settings = nil
		return
	}

	s.settings = make(map[string]any, len(settings))
	for key, value := range settings {
		s.settings[key] = value
	}
}

func (s *Suite) LintCode(lines ...string) {
	s.T().Helper()

	path := s.nextCodePath()
	s.GivenFile(path, lines...)
	s.LintFile(path)
}

func (s *Suite) FixCode(lines ...string) {
	s.T().Helper()

	path := s.nextCodePath()
	s.GivenFile(path, lines...)
	s.FixFile(path)
}

func (s *Suite) GivenFile(path string, lines ...string) {
	s.T().Helper()

	if path == "" {
		s.T().Fatal("file path is required")
	}

	if s.files == nil {
		s.files = make(map[string]string)
	}

	s.files[path] = joinLines(lines...)
}

func (s *Suite) LintFile(path string) {
	s.T().Helper()

	s.run(path, false)
}

func (s *Suite) FixFile(path string) {
	s.T().Helper()

	s.run(path, true)
}

func (s *Suite) ShouldPass() {
	s.T().Helper()

	result := s.requireResult()
	if result.ExitCode != 0 {
		s.T().Fatalf(
			"expected passing run, got exit code %d\nerror:\n%s\nstdout:\n%s\nstderr:\n%s",
			result.ExitCode,
			result.Error,
			result.Stdout,
			result.Stderr,
		)
	}

	if !isEmptyOutput(result.Output) {
		s.T().Fatalf("expected no output, got:\n%s", result.Output)
	}
}

func (s *Suite) ShouldFailWith(fragments ...string) {
	s.T().Helper()

	result := s.requireResult()
	if result.ExitCode == 0 {
		s.T().Fatalf("expected failing run, got exit code 0\noutput:\n%s", result.Output)
	}

	for _, fragment := range fragments {
		if strings.Contains(result.Output, fragment) {
			continue
		}

		s.T().Fatalf("missing output fragment %q\nfull output:\n%s", fragment, result.Output)
	}
}

func (s *Suite) ShouldProduce(lines ...string) {
	s.T().Helper()

	result := s.requireResult()
	if len(result.FixedFiles) == 0 {
		s.T().Fatal("no fixed output available yet; call FixFile or FixCode first")
	}

	if s.lastTargetPath == "" {
		s.T().Fatal("no target file available for fixed output assertion")
	}

	actual, ok := result.FixedFiles[s.lastTargetPath]
	if !ok {
		s.T().Fatalf("expected fixed file %q but it was not written", s.lastTargetPath)
	}

	expected := joinLines(lines...)
	if actual != expected {
		s.T().Fatalf("fixed file %q mismatch:\nexpected:\n%s\nactual:\n%s", s.lastTargetPath, expected, actual)
	}
}

func (s *Suite) run(path string, fix bool) {
	s.T().Helper()

	workspace := newWorkspace(s.T())
	scenario := scenarioInput{
		ModulePath: s.ModulePath,
		GoVersion:  s.GoVersion,
		Linter:     s.Linter,
		Files:      copyStringMap(s.files),
		Settings:   copyAnyMap(s.settings),
	}

	writeWorkspace(s.T(), workspace, scenario)

	if !pathExistsIn(path, scenario.Files) {
		s.T().Fatalf("target file %q is not defined in this test", path)
	}

	var result Result
	if fix {
		result = runCustomGCLFix(s.T(), workspace, s.Linter)
		result.FixedFiles = readFixedFiles(s.T(), workspace, scenario.Files)
	} else {
		result = runCustomGCL(s.T(), workspace, s.Linter)
	}

	s.workspace = workspace
	s.lastTargetPath = path
	s.lastResult = &result
}

func (s *Suite) nextCodePath() string {
	path := fmt.Sprintf("lint%d.go", s.codeCounter)
	s.codeCounter++
	return path
}

func (s *Suite) requireResult() Result {
	s.T().Helper()

	if s.lastResult == nil {
		s.T().Fatal("no result available yet; call LintFile, LintCode, FixFile, or FixCode first")
	}

	return *s.lastResult
}

func newWorkspace(t testing.TB) string {
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

func runCustomGCL(t testing.TB, workspace string, _ string) Result {
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

	return runCommand(t, cmd, workspace, "running custom-gcl")
}

func runCustomGCLFix(t testing.TB, workspace string, _ string) Result {
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

	return runCommand(t, cmd, workspace, "running custom-gcl --fix")
}

func runCommand(t testing.TB, cmd *exec.Cmd, workspace string, fatalPrefix string) Result {
	t.Helper()

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

	t.Fatalf("%s: %v", fatalPrefix, err)
	return Result{}
}

func readFixedFiles(t testing.TB, workspace string, files map[string]string) map[string]string {
	t.Helper()

	fixed := make(map[string]string)
	for path := range files {
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

func acquireCustomGCLLock(t testing.TB) func() {
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

func writeWorkspace(t testing.TB, workspace string, scenario scenarioInput) {
	t.Helper()

	if scenario.Linter == "" {
		t.Fatal("suite linter is required")
	}

	if len(scenario.Files) == 0 {
		t.Fatal("at least one file is required")
	}

	files := copyStringMap(scenario.Files)
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

func renderConfig(scenario scenarioInput) string {
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

func renderGoMod(scenario scenarioInput) string {
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

type scenarioInput struct {
	ModulePath string
	GoVersion  string
	Linter     string
	Files      map[string]string
	Settings   map[string]any
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

func joinLines(lines ...string) string {
	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n"
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func copyAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func pathExistsIn(path string, files map[string]string) bool {
	_, ok := files[path]
	return ok
}
