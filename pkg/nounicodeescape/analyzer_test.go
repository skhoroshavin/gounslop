package nounicodeescape_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/pkg/nounicodeescape"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, nounicodeescape.Analyzer, "a")
}
