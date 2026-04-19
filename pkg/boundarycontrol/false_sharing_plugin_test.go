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
	s.ShouldFailWith("boundarycontrol", "invalid architecture settings", "shared")
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
		"import _ \"example.com/mod/shared/lib/http\"",
	)
	s.GivenFile("shared/lib/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldFailWith("shared/lib/http/http.go", "only used by: feature/api", "Must be used by 2+ entities")
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
		"import _ \"example.com/mod/shared/contracts/http\"",
	)
	s.GivenFile("shared/contracts/http/http.go",
		"package http",
		"",
		"var X = 1",
	)
	s.LintFile("feature/api/consumer.go")
	s.ShouldFailWith("shared/contracts/http/http.go", "only used by: feature/api", "Must be used by 2+ entities")
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
		"import _ \"example.com/mod/shared/lib/http\"",
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
		"import _ \"example.com/mod/shared\"",
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
	s.ShouldFailWith("shared/util.go", "only used by: featurea", "Must be used by 2+ entities")
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
		"import _ \"example.com/mod/shared\"",
	)
	s.GivenFile("featureb/consumer.go",
		"package featureb",
		"",
		"import _ \"example.com/mod/shared\"",
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
		"import _ \"example.com/mod/shared\"",
	)
	s.GivenFile("featurea/consumer_test.go",
		"package featurea",
		"",
		"import _ \"example.com/mod/shared\"",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer.go")
	s.ShouldFailWith("shared/util.go", "only used by: featurea", "Must be used by 2+ entities")
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
	s.ShouldFailWith("shared/util.go", "not imported by any entity", "Must be used by 2+ entities")
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
		"import _ \"example.com/mod/shared\"",
	)
	s.GivenFile("featurea/consumer_b.go",
		"package featurea",
		"",
		"import _ \"example.com/mod/shared\"",
	)
	s.GivenFile("shared/util.go",
		"package shared",
		"",
		"func Value() string { return \"x\" }",
	)
	s.LintFile("featurea/consumer_a.go")
	s.ShouldFailWith("shared/util.go", "only used by: featurea", "Must be used by 2+ entities")
}
