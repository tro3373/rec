# Tests and coverage.

pkg := ./...
cover_mode := atomic
cover_out := cover.out

test: testsum-cover-check
# test-normal:
# 	@echo "==> Testing $(pkg)" >&2
# 	@go test -v $(pkg)
# test-cover:
# 	@echo "==> Running go test with coverage check" >&2
# 	@go test $(pkg) -coverprofile=$(cover_out) -covermode=$(cover_mode) -coverpkg=$(pkg)
# test-cover-count:
# 	@echo "==> Running go test with coverage check (count mode)" >&2
# 	@make test-cover cover_mode=count
# 	@go tool cover -func=$(cover_out)
# test-cover-html: test-cover
# 	@go tool cover -html=$(cover_out) -o cover.html
# test-cover-open: test-cover
# 	@go tool cover -html=$(cover_out)
# test-cover-check: test-cover-html
# 	@echo "==> Checking coverage threshold" >&2
# 	@go-test-coverage --config=./.testcoverage.yml
testsum:
	@echo "==> Running go testsum" >&2
	@gotestsum --format testname -- -v $(pkg) -coverprofile=$(cover_out) -covermode=$(cover_mode) -coverpkg=$(pkg)
testsum-cover-check: testsum
	@echo "==> Running test-coverage" >&2
	@go-test-coverage --config=./.testcoverage.yaml
