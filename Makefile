pkgs := $(shell go list ./...)
run := .
count := 1
timeout := 120m

test: 
	KNUU_TIMEOUT=120m go test -v $(pkgs) -run $(run) -count=$(count) -timeout $(timeout)
.PHONY: test
