package boundarycontrol_test

func (s *BoundarycontrolE2ESuite) TestRemovedModeSettingFailsClearly() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
				"mode":   "dir",
			},
		},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"func Use() {}",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldFailWith("boundarycontrol", "architecture[\"shared\"].mode is unsupported")
}

func (s *BoundarycontrolE2ESuite) TestSharedFlagWrongTypeFailsClearly() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": "yes",
			},
		},
	})
	s.GivenFile("feature/consumer.go",
		"package feature",
		"",
		"func Use() {}",
	)
	s.LintFile("feature/consumer.go")
	s.ShouldFailWith("gounslop", "invalid settings", "shared")
}

func (s *BoundarycontrolE2ESuite) TestExportsWrongTypeFailsClearly() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"pkg/api": map[string]any{
				"exports": "^New[A-Z].*$",
			},
		},
	})
	s.GivenFile("pkg/api/api.go",
		"package api",
		"",
		"func NewClient() {}",
	)
	s.LintFile("pkg/api/api.go")
	s.ShouldFailWith("gounslop", "invalid settings", "exports")
}

func (s *BoundarycontrolE2ESuite) TestInvalidExportRegexFailsClearly() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"pkg/api": map[string]any{
				"exports": []string{"("},
			},
		},
	})
	s.GivenFile("pkg/api/api.go",
		"package api",
		"",
		"func NewClient() {}",
	)
	s.LintFile("pkg/api/api.go")
	s.ShouldFailWith("boundarycontrol", `architecture["pkg/api"].exports[0]: invalid regex`)
}

func (s *BoundarycontrolE2ESuite) TestExportContractsAllowMatchingTopLevelDeclarations() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"pkg/api": map[string]any{
				"exports": []string{"^New[A-Z].*$", "^Client$"},
			},
		},
	})
	s.GivenFile("pkg/api/api.go",
		"package api",
		"",
		"type Client struct{}",
		"",
		"func NewClient() Client {",
		"\treturn Client{}",
		"}",
		"",
		"func buildClient() Client {",
		"\treturn Client{}",
		"}",
		"",
		"func (Client) Build() Client {",
		"\treturn Client{}",
		"}",
	)
	s.LintFile("pkg/api/api.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestExportContractsReportViolatingTopLevelDeclaration() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"pkg/api": map[string]any{
				"exports": []string{"^New[A-Z].*$"},
			},
		},
	})
	s.GivenFile("pkg/api/api.go",
		"package api",
		"",
		"func BuildClient() {}",
	)
	s.LintFile("pkg/api/api.go")
	s.ShouldFailWith("pkg/api/api.go", "BuildClient does not match boundarycontrol export contract")
}

func (s *BoundarycontrolE2ESuite) TestExportContractsUseFullNameMatching() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"pkg/api": map[string]any{
				"exports": []string{"Error"},
			},
		},
	})
	s.GivenFile("pkg/api/api.go",
		"package api",
		"",
		"type ClientError struct{}",
	)
	s.LintFile("pkg/api/api.go")
	s.ShouldFailWith("pkg/api/api.go", "ClientError does not match boundarycontrol export contract")
}

func (s *BoundarycontrolE2ESuite) TestExactSharedSelectorMarksSubtreeAsShared() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared/lib": map[string]any{
				"shared": true,
			},
			"feature/api": map[string]any{
				"imports": []string{"shared/lib/http"},
			},
		},
	})
	s.GivenFile("feature/api/consumer.go",
		"package api",
		"",
		"import http \"example.com/mod/shared/lib/http\"",
		"",
		"var _ = http.X",
	)
	s.GivenFile("shared/lib/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldFailWith("shared/lib/http/http.go", "X only used by: feature/api", "Must be used by 2+ entities")
}

func (s *BoundarycontrolE2ESuite) TestWildcardSharedSelectorMarksDirectChildSubtreeAsShared() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared/*": map[string]any{
				"shared": true,
			},
			"feature/api": map[string]any{
				"imports": []string{"shared/contracts/http"},
			},
		},
	})
	s.GivenFile("feature/api/consumer.go",
		"package api",
		"",
		"import http \"example.com/mod/shared/contracts/http\"",
		"",
		"var _ = http.X",
	)
	s.GivenFile("shared/contracts/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldFailWith("shared/contracts/http/http.go", "X only used by: feature/api", "Must be used by 2+ entities")
}

func (s *BoundarycontrolE2ESuite) TestSelectorWithoutSharedFlagIsNotSharedDeclaration() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared/lib": map[string]any{},
			"feature/api": map[string]any{
				"imports": []string{"shared/lib/http"},
			},
		},
	})
	s.GivenFile("feature/api/consumer.go",
		"package api",
		"",
		"import http \"example.com/mod/shared/lib/http\"",
		"",
		"var _ = http.X",
	)
	s.GivenFile("shared/lib/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestSharedPackageWithSingleConsumerFails() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
			},
			"featurea": map[string]any{
				"imports": []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"func Use() {}",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("shared/util.go", "Value only used by: featurea", "Must be used by 2+ entities")
}

func (s *BoundarycontrolE2ESuite) TestSharedPackageWithMultipleConsumersPasses() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
			},
			"featurea": map[string]any{
				"imports": []string{"shared"},
			},
			"featureb": map[string]any{
				"imports": []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestSharedPackageTestFilesDoNotIncreaseConsumerCount() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
			},
			"featurea": map[string]any{
				"imports": []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("featurea/consumer_test.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("shared/util.go", "Value only used by: featurea", "Must be used by 2+ entities")
}

func (s *BoundarycontrolE2ESuite) TestSharedPackageWithNoConsumersFails() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
			},
			"featurea": map[string]any{
				"imports": []string{},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"func Use() {}",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("shared/util.go", "Value not used by any entity", "Must be used by 2+ entities")
}

func (s *BoundarycontrolE2ESuite) TestTwoFilesInSamePackageCountAsOneConsumer() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
			},
			"featurea": map[string]any{
				"imports": []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer_a.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("featurea/consumer_b.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer_a.go")
	s.ShouldFailWith("shared/util.go", "Value only used by: featurea", "Must be used by 2+ entities")
}

func (s *BoundarycontrolE2ESuite) TestDifferentSymbolsUsedByDifferentConsumersFailSeparately() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
			},
			"featurea": map[string]any{
				"imports": []string{"shared"},
			},
			"featureb": map[string]any{
				"imports": []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.A()",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.B()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func A() string { return \"a\" }",
		"func B() string { return \"b\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith(
		"shared/util.go",
		"A only used by: featurea",
		"B only used by: featureb",
		"Must be used by 2+ entities",
	)
}

func (s *BoundarycontrolE2ESuite) TestInternalSharedPackageReferenceCountsAsConsumer() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
			},
			"featurea": map[string]any{
				"imports": []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.Value()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
		"func useValue() string { return Value() }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestSharedPackageWithNoExportedSymbolsProducesNoDiagnostics() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
			},
			"featurea": map[string]any{
				"imports": []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import _ \"example.com/mod/shared\"",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldPass()
}

func (s *BoundarycontrolE2ESuite) TestExportedSymbolFormsUsingTypeInfo() {
	s.GivenConfig(map[string]any{
		"architecture": map[string]any{
			"shared": map[string]any{
				"shared": true,
			},
			"featurea": map[string]any{
				"imports": []string{"shared"},
			},
		},
	})
	s.GivenFile("featurea/consumer.go",
		"package featurea",
		"",
		"import \"example.com/mod/shared\"",
		"",
		"var _ = shared.FuncValue()",
		"var _ shared.Widget",
		"var _ = shared.SharedValue",
		"const _ = shared.SharedConst",
		"var _ = shared.Worker{}.Do()",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func FuncValue() string { return \"x\" }",
		"type Widget struct{}",
		"var SharedValue = 1",
		"const SharedConst = \"x\"",
		"type Worker struct{}",
		"func (Worker) Do() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith(
		"shared/util.go",
		"FuncValue only used by: featurea",
		"Widget only used by: featurea",
		"SharedValue only used by: featurea",
		"SharedConst only used by: featurea",
		"Worker.Do only used by: featurea",
		"Must be used by 2+ entities",
	)
}
