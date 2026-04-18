package readfriendlyorder_test

import (
	"testing"

	"github.com/skhoroshavin/gounslop/internal/ruletest"
	"github.com/stretchr/testify/suite"
)

const readfriendlyorderModulePath = "example.com/mod"

type ReadfriendlyorderE2ESuite struct {
	suite.Suite
}

func TestPluginE2E(t *testing.T) {
	suite.Run(t, new(ReadfriendlyorderE2ESuite))
}

func (s *ReadfriendlyorderE2ESuite) TestCorrectTopLevelOrderPasses() {
	s.runScenario(ruletest.Scenario{
		Name:       "correct top-level order passes",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"valid.go": "package main\n\nfunc Exported() int {\n\treturn helper() + constant\n}\n\nfunc helper() int { return 1 }\n\nconst constant = 42\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestIncorrectTopLevelOrderFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "incorrect top-level order flagged",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"invalid.go": "package main\n\nfunc helperBad() int { return 1 }\n\nfunc ExportedBad() int {\n\treturn helperBad()\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"Place helper",
				"ExportedBad",
			},
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestConstantBeforeFunctionFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "constant before function flagged",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"invalid_const.go": "package main\n\nconst maxCount = 3\n\nfunc Limit() int {\n\treturn maxCount\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"Place constant",
				"Limit",
			},
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestMethodOrderingFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "method ordering flagged",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"invalid.go": "package main\n\ntype Worker struct{}\n\nfunc (w *Worker) doWork() int { return 1 }\n\nfunc (w *Worker) Process() int {\n\treturn w.doWork()\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"Place method",
				"doWork",
				"Process",
			},
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestConstructorPlacementFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "constructor placement flagged",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"constructor.go": "package main\n\ntype Handler struct{}\n\nfunc (h *Handler) Handle() int { return 1 }\n\nfunc NewHandler() *Handler { return &Handler{} }\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"Place constructor",
				"NewHandler",
				"Handler",
			},
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestInitOrderingFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "init ordering flagged",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"init.go": "package main\n\nfunc Setup() {}\n\nfunc init() {}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"Place init()",
				"Setup",
			},
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestCyclicDependenciesExempt() {
	s.runScenario(ruletest.Scenario{
		Name:       "cyclic dependencies exempt",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"cyclic.go": "package main\n\nfunc parseExpression() int {\n\treturn parseAtom()\n}\n\nfunc parseAtom() int {\n\treturn parseExpression()\n}\n\nfunc Parse() int {\n\treturn parseExpression()\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestValidMethodOrderPasses() {
	s.runScenario(ruletest.Scenario{
		Name:       "valid method order passes",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"methods.go": "package main\n\ntype Service struct{}\n\nfunc NewService() *Service { return &Service{} }\n\nfunc (s *Service) Run() int {\n\treturn s.compute()\n}\n\nfunc (s *Service) compute() int {\n\treturn 1\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestEagerEvaluationExempt() {
	s.runScenario(ruletest.Scenario{
		Name:       "eager evaluation: constant referenced by another constant is exempt",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"eager.go": "package main\n\nconst pi = 3.14\n\nconst twicePi = pi * 2\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestTestMainOrderingFlagged() {
	s.runScenario(ruletest.Scenario{
		Name:       "TestMain ordering flagged in test file",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"main_test.go": "package main\n\nimport \"testing\"\n\nfunc TestSomething(t *testing.T) {}\n\nfunc TestMain(m *testing.M) {}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode: 1,
			OutputContains: []string{
				"Place TestMain",
			},
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestHelperBeforeExportedFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "helper before exported is reordered by fix",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"fix.go": "package main\n\nfunc helper() int { return 1 }\n\nfunc Exported() int {\n\treturn helper()\n}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"fix.go": "package main\n\nfunc Exported() int {\n\treturn helper()\n}\n\nfunc helper() int { return 1 }\n",
			},
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestInitOrderingFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "init ordering fix swaps init before exported func",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"init.go": "package main\n\nfunc Setup() {}\n\nfunc init() {}\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"init.go": "package main\n\nfunc init() {}\n\nfunc Setup() {}\n",
			},
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) TestConstructorPlacementFix() {
	s.runFixScenario(ruletest.Scenario{
		Name:       "constructor placement fix moves constructor after type",
		ModulePath: readfriendlyorderModulePath,
		Linter:     "readfriendlyorder",
		Files: map[string]string{
			"constructor.go": "package main\n\ntype Handler struct{}\n\nfunc (h *Handler) Handle() int { return 1 }\n\nfunc NewHandler() *Handler { return &Handler{} }\n",
		},
		Expect: ruletest.Expectation{
			ExitCode:    0,
			EmptyOutput: true,
			FixedFiles: map[string]string{
				"constructor.go": "package main\n\ntype Handler struct{}\n\nfunc NewHandler() *Handler { return &Handler{} }\n\nfunc (h *Handler) Handle() int { return 1 }\n",
			},
		},
	})
}

func (s *ReadfriendlyorderE2ESuite) runScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.Execute(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}

func (s *ReadfriendlyorderE2ESuite) runFixScenario(scenario ruletest.Scenario) {
	s.T().Helper()

	result := ruletest.ExecuteFix(s.T(), scenario)
	ruletest.AssertResult(s.T(), scenario.Expect, result)
}
