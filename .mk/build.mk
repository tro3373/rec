# Building and running the binary.

.PHONY: clean clean-cache build run install service \
	build-linux-arm build-linux-amd build-android-arm build-android-amd \
	build-darwin-arm build-darwin-amd build-windows-arm build-windows-amd

dst := ./rec
main_pkg := ./cmd/rec

clean:
	@echo "==> Removing $(dst)" >&2
	@rm -f $(dst)

# Separate from `clean`, which every build goes through: these caches are shared
# by every Go module on the machine, so dropping them is never free.
clean-cache:
	@echo "==> Dropping the Go build and test caches" >&2
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
env_file := $(or $(XDG_CONFIG_HOME),$(HOME)/.config)/rec/env

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
