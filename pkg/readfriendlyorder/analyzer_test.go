package readfriendlyorder_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/pkg/readfriendlyorder"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestTopLevelOrdering(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, readfriendlyorder.Analyzer, "toplevel")
}

func TestCyclicHelpers(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, readfriendlyorder.Analyzer, "cyclic")
}

func TestEagerEvaluation(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, readfriendlyorder.Analyzer, "eager")
}

func TestMethodOrdering(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, readfriendlyorder.Analyzer, "methods")
}

func TestInitOrdering(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, readfriendlyorder.Analyzer, "initorder")
}

func TestTestOrdering(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, readfriendlyorder.Analyzer, "testorder")
}
