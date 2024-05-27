pkgs := $(shell go list ./...)
run := .
count := 1

test: 
	KNUU_TIMEOUT=120m go test -v $(pkgs) -run $(run) -count=$(count) -timeout 120m
.PHONY: test
