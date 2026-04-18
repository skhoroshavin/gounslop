.PHONY: lint test

custom-gcl: $(shell find . -name '*.go' -not -path './testdata/*') .custom-gcl.yml go.mod go.sum
	golangci-lint custom
	@touch custom-gcl

lint: custom-gcl
	./custom-gcl run ./...

test: custom-gcl
	go test ./...
