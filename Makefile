pkgs := $(shell go list ./...)
run := .
count := 1
timeout := 120m

## help: Show this help message
help: Makefile
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
.PHONY: help

## clean: clean testcache
clean:
	@echo "--> Clearing testcache"
	@go clean --testcache
	@rm coverage.txt
.PHONY: clean

## cover: generate to code coverage report.
cover:
	@echo "--> Generating Code Coverage"
	@go install github.com/ory/go-acc@latest
	@go-acc -o coverage.txt $(pkgs)
.PHONY: cover

## deps: Install dependencies
deps:
	@echo "--> Installing dependencies"
	@go mod download
	@go mod tidy
.PHONY: deps

## lint: Run linters golangci-lint and markdownlint.
lint: vet
	@echo "--> Running markdownlint"
	@markdownlint --config .markdownlint.yaml '**/*.md'
	@echo "--> Running yamllint"
	@yamllint --no-warnings . -c .yamllint.yml
	@echo "--> Running actionlint"
	@actionlint
	@echo "--> Running govulncheck"
	@govulncheck ./...
	@echo "--> Running golangci-lint"
	@golangci-lint run
.PHONY: lint

## fmt: Run fixes for linters.
fmt:
	@echo "--> Formatting markdownlint"
	@markdownlint --config .markdownlint.yaml '**/*.md' -f
	@echo "--> Formatting go"
	@golangci-lint run --fix
.PHONY: fmt

## vet: Run go vet
vet: 
	@echo "--> Running go vet"
	@go vet $(pkgs)
.PHONY: vet

## test: Run the testsuite
test: vet 
	go test -v $(pkgs) -run $(run) -count=$(count) -timeout $(timeout)
.PHONY: test
