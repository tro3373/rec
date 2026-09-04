# Static analysis.

.PHONY: lint lint-all

lint:
	@echo "==> Running golangci-lint run" >&2
	@golangci-lint run
# The config limits findings to the diff against HEAD. Use this to sweep everything.
lint-all:
	@echo "==> Running golangci-lint run --new=false" >&2
	@golangci-lint run --new=false
