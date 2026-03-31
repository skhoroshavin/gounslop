package nofalsesharing_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/skhoroshavin/gounslop/pkg/nofalsesharing"
)

// --- file-mode tests ---

func TestFileMode_SharedByTwoConsumers_NoError(t *testing.T) {
	dir := testdataDir("testmod")
	diags, err := nofalsesharing.Run(dir, "example.com/testmod", []nofalsesharing.DirConfig{
		{Path: "shared", Mode: nofalsesharing.FileMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestFileMode_SharedBySingleConsumer_Error(t *testing.T) {
	dir := testdataDir("testmod_single")
	diags, err := nofalsesharing.Run(dir, "example.com/testmod_single", []nofalsesharing.DirConfig{
		{Path: "shared", Mode: nofalsesharing.FileMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "only used by:") {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}
	if !strings.Contains(diags[0].Message, "Must be used by 2+ entities") {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}
}

func TestFileMode_TestFilesDontCountAsConsumers(t *testing.T) {
	dir := testdataDir("testmod_testfiles")
	diags, err := nofalsesharing.Run(dir, "example.com/testmod_testfiles", []nofalsesharing.DirConfig{
		{Path: "shared", Mode: nofalsesharing.FileMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Only 1 non-test consumer (featureA/consumer.go), test files don't count
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic (test files don't count), got %d: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "only used by:") {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}
}

func TestFileMode_NotImportedByAnyone_Error(t *testing.T) {
	dir := testdataDir("testmod_noConsumer")
	diags, err := nofalsesharing.Run(dir, "example.com/testmod_noconsumer", []nofalsesharing.DirConfig{
		{Path: "shared", Mode: nofalsesharing.FileMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "not imported by any entity") {
		t.Errorf("expected 'not imported by any entity', got: %s", diags[0].Message)
	}
}

// --- dir-mode tests ---

func TestDirMode_TwoDifferentDirs_NoError(t *testing.T) {
	dir := testdataDir("testmod")
	diags, err := nofalsesharing.Run(dir, "example.com/testmod", []nofalsesharing.DirConfig{
		{Path: "shared", Mode: nofalsesharing.DirMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for dir mode with 2 dirs, got %d: %v", len(diags), diags)
	}
}

func TestDirMode_SameDir_Error(t *testing.T) {
	dir := testdataDir("testmod_single")
	diags, err := nofalsesharing.Run(dir, "example.com/testmod_single", []nofalsesharing.DirConfig{
		{Path: "shared", Mode: nofalsesharing.DirMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "only used by: featureA") {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}
}

func TestDirMode_TwoFilesInSameDir_Error(t *testing.T) {
	// In dir mode, two files in the same package/directory count as 1 consumer
	dir := testdataDir("testmod_dirmode_single")
	diags, err := nofalsesharing.Run(dir, "example.com/testmod_dirmode_single", []nofalsesharing.DirConfig{
		{Path: "shared", Mode: nofalsesharing.DirMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic (same dir = 1 consumer), got %d: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "only used by: featureA") {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}
}

func TestDirMode_TwoFilesInSameDir_FileMode_NoError(t *testing.T) {
	// In file mode, two files in the same dir count as 2 consumers
	dir := testdataDir("testmod_dirmode_single")
	diags, err := nofalsesharing.Run(dir, "example.com/testmod_dirmode_single", []nofalsesharing.DirConfig{
		{Path: "shared", Mode: nofalsesharing.FileMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics (2 files = 2 consumers in file mode), got %d: %v", len(diags), diags)
	}
}

func TestDirMode_TestFilesDontCountAsConsumers(t *testing.T) {
	dir := testdataDir("testmod_testfiles")
	diags, err := nofalsesharing.Run(dir, "example.com/testmod_testfiles", []nofalsesharing.DirConfig{
		{Path: "shared", Mode: nofalsesharing.DirMode},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Only featureA has a non-test consumer, featureB only has test file
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic (test files don't count in dir mode), got %d: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "only used by: featureA") {
		t.Errorf("unexpected message: %s", diags[0].Message)
	}
}

func testdataDir(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", name)
}
