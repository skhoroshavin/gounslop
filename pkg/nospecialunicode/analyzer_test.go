package nospecialunicode_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/pkg/nospecialunicode"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, nospecialunicode.Analyzer, "a")
}
