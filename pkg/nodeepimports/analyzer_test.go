package nodeepimports_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/pkg/nodeepimports"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	if err := nodeepimports.Analyzer.Flags.Set("module-root", "example.com/mod"); err != nil {
		t.Fatal(err)
	}
	analysistest.Run(t, testdata, nodeepimports.Analyzer,
		"example.com/mod/a",
		"example.com/mod/a/child",
	)
}
