# Building and running the binary.

dst := ./rec
main_pkg := ./cmd/rec

clean:
	@echo "==> Cleaning" >&2
	@rm -f $(dst)
	@go clean -cache -testcache

build: build-linux-amd
build-linux-arm: _build-linux-arm64
build-linux-amd: _build-linux-amd64
build-android-arm: _build-android-arm64
build-android-amd: _build-android-amd64
build-darwin-arm: _build-darwin-arm64
build-darwin-amd: _build-darwin-amd64
build-windows-arm: _build-windows-arm64
build-windows-amd: _build-windows-amd64
# CGO_ENABLED=0: Disable CGO
# -trimpath: Remove all file system paths from the resulting executable.
# -s: Omit the symbol table and debug information.
# -w: Omit the DWARF symbol table.
_build-%: clean lint
	@echo "==> Go Building" >&2
	$(eval goos=$(firstword $(subst -, ,$*)))
	$(eval goarch=$(word 2, $(subst -, ,$*)))
	$(eval mainver=$(shell ver))
	@env GOOS=$(goos) GOARCH=$(goarch) CGO_ENABLED=0 \
		go build -v \
			-trimpath \
			-ldflags="-s -w -X main.version=$(mainver)" \
			-o $(dst) \
			$(main_pkg)
run:
	@go run $(main_pkg) $(ARGS)
