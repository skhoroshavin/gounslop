package tests

import (
	"testing"

	"github.com/skhoroshavin/gounslop/tests/rule"
	"github.com/stretchr/testify/suite"
)

type ReadfriendlyorderE2ESuite struct {
	rule.Suite
}

func (s *ReadfriendlyorderE2ESuite) SetupTest() {
	s.Suite.SetupTest()
	s.ModulePath = "example.com/mod"
}

func TestReadfriendlyorderE2E(t *testing.T) {
	suite.Run(t, new(ReadfriendlyorderE2ESuite))
}

func (s *ReadfriendlyorderE2ESuite) TestCorrectTopLevelOrderPasses() {
	s.LintCode(
		"package main",
		"",
		"func Exported() int {",
		"\treturn helper() + constant",
		"}",
		"",
		"func helper() int { return 1 }",
		"",
		"const constant = 42",
	)
	s.ShouldPass()
}

func (s *ReadfriendlyorderE2ESuite) TestIncorrectTopLevelOrderFlagged() {
	s.LintCode(
		"package main",
		"",
		"func helperBad() int { return 1 }",
		"",
		"func ExportedBad() int {",
		"\treturn helperBad()",
		"}",
	)
	s.ShouldFailWith("Place helper", "ExportedBad")
}

func (s *ReadfriendlyorderE2ESuite) TestConstantBeforeFunctionFlagged() {
	s.LintCode(
		"package main",
		"",
		"const maxCount = 3",
		"",
		"func Limit() int {",
		"\treturn maxCount",
		"}",
	)
	s.ShouldFailWith("Place constant", "Limit")
}

func (s *ReadfriendlyorderE2ESuite) TestMethodOrderingFlagged() {
	s.LintCode(
		"package main",
		"",
		"type Worker struct{}",
		"",
		"func (w *Worker) doWork() int { return 1 }",
		"",
		"func (w *Worker) Process() int {",
		"\treturn w.doWork()",
		"}",
	)
	s.ShouldFailWith("Place method", "doWork", "Process")
}

func (s *ReadfriendlyorderE2ESuite) TestConstructorPlacementFlagged() {
	s.LintCode(
		"package main",
		"",
		"type Handler struct{}",
		"",
		"func (h *Handler) Handle() int { return 1 }",
		"",
		"func NewHandler() *Handler { return &Handler{} }",
	)
	s.ShouldFailWith("Place constructor", "NewHandler", "Handler")
}

func (s *ReadfriendlyorderE2ESuite) TestInitOrderingFlagged() {
	s.LintCode(
		"package main",
		"",
		"func Setup() {}",
		"",
		"func init() {}",
	)
	s.ShouldFailWith("Place init()", "Setup")
}

func (s *ReadfriendlyorderE2ESuite) TestCyclicDependenciesExempt() {
	s.LintCode(
		"package main",
		"",
		"func parseExpression() int {",
		"\treturn parseAtom()",
		"}",
		"",
		"func parseAtom() int {",
		"\treturn parseExpression()",
		"}",
		"",
		"func Parse() int {",
		"\treturn parseExpression()",
		"}",
	)
	s.ShouldPass()
}

func (s *ReadfriendlyorderE2ESuite) TestValidMethodOrderPasses() {
	s.LintCode(
		"package main",
		"",
		"type Service struct{}",
		"",
		"func NewService() *Service { return &Service{} }",
		"",
		"func (s *Service) Run() int {",
		"\treturn s.compute()",
		"}",
		"",
		"func (s *Service) compute() int {",
		"\treturn 1",
		"}",
	)
	s.ShouldPass()
}

func (s *ReadfriendlyorderE2ESuite) TestEagerEvaluationExempt() {
	s.LintCode(
		"package main",
		"",
		"const pi = 3.14",
		"",
		"const twicePi = pi * 2",
	)
	s.ShouldPass()
}

func (s *ReadfriendlyorderE2ESuite) TestTestMainOrderingFlagged() {
	s.GivenFile("main_test.go",
		"package main",
		"",
		"import \"testing\"",
		"",
		"func TestSomething(t *testing.T) {}",
		"",
		"func TestMain(m *testing.M) {}",
	)
	s.LintFile("main_test.go")
	s.ShouldFailWith("Place TestMain")
}

func (s *ReadfriendlyorderE2ESuite) TestHelperBeforeExportedFix() {
	s.FixCode(
		"package main",
		"",
		"func helper() int { return 1 }",
		"",
		"func Exported() int {",
		"\treturn helper()",
		"}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func Exported() int {",
		"\treturn helper()",
		"}",
		"",
		"func helper() int { return 1 }",
	)
}

func (s *ReadfriendlyorderE2ESuite) TestInitOrderingFix() {
	s.FixCode(
		"package main",
		"",
		"func Setup() {}",
		"",
		"func init() {}",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"func init() {}",
		"",
		"func Setup() {}",
	)
}

func (s *ReadfriendlyorderE2ESuite) TestConstructorPlacementFix() {
	s.FixCode(
		"package main",
		"",
		"type Handler struct{}",
		"",
		"func (h *Handler) Handle() int { return 1 }",
		"",
		"func NewHandler() *Handler { return &Handler{} }",
	)
	s.ShouldPass()
	s.ShouldProduce(
		"package main",
		"",
		"type Handler struct{}",
		"",
		"func NewHandler() *Handler { return &Handler{} }",
		"",
		"func (h *Handler) Handle() int { return 1 }",
	)
}
