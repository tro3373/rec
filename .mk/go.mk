# Go module housekeeping.

.PHONY: tidy tidy-go deps update fmt

tidy:
	@echo "==> Running go mod tidy -v"
	@go mod tidy -v
tidy-go:
	@v=$(shell go version|awk '{print $$3}' |sed -e 's,go\(.*\)\..*,\1,g') && go mod tidy -go=$${v}
deps:
	@go list -m all
update:
	@go get -u ./...
fmt:
	@echo "==> Running go fmt ./..." >&2
	@go fmt ./...
