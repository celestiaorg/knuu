pkgs := $(shell go list ./...)
run := .
count := 1


test: 
	go test -v $(pkgs) -run $(run) -count=$(count) 
.PHONY: test
