GO ?= go
TEST_PATH ?= ./...
BENCH ?= BenchmarkParseFrameworks

.PHONY: help test vet bench race deps clean

help:
	@echo "Targets: test vet bench race deps clean"

test:
	$(GO) test $(TEST_PATH)

vet:
	$(GO) vet ./...

bench:
	$(GO) test ./... -run '^$$' -bench $(BENCH) -count 6

race:
	$(GO) test -race ./...

deps:
	$(GO) mod tidy

clean:
	$(GO) clean ./...

