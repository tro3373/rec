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

# Override any of these to install somewhere else.
bin_dir := $(HOME)/.local/bin
service_dir := $(HOME)/.config/systemd/user
env_file := $(HOME)/.config/rec/env

# Everything the service needs, short of starting it. See `service`.
install: build
	@echo "==> Installing $(bin_dir)/rec" >&2
	@install -Dm755 $(dst) $(bin_dir)/rec
	@echo "==> Installing $(service_dir)/rec.service" >&2
	@install -Dm644 systemd/rec.service $(service_dir)/rec.service
	@if [[ -f "$(env_file)" ]]; then \
		echo "==> Env file already present: $(env_file)" >&2; \
	else \
		echo "==> Writing a starter env file: $(env_file)" >&2; \
		install -Dm600 systemd/env.example "$(env_file)"; \
	fi
	@systemctl --user daemon-reload
	@echo "==> Installed. Next:" >&2
	@echo "      1. edit $(env_file)" >&2
	@echo "      2. make service" >&2

# Enables the unit and picks up a freshly installed binary.
service:
	@echo "==> Enabling and starting rec" >&2
	@systemctl --user enable rec
	@systemctl --user restart rec
	@systemctl --user --no-pager --lines=0 status rec
