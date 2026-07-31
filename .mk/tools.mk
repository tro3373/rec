# Installing everything rec needs to run and to be developed.
#
# Runtime:
#   ffmpeg      records the two tracks
#   pactl       resolves the default input and monitor sources (libpulse)
#   whisper-cli local transcription (whisper-cpp), only for -engine whisper
#   claude      generates the minutes; install it yourself, no distro package
#
# NOTE: whisper_model mirrors defaultWhisperModel in internal/rec/whisper.go.
#       Keep both in sync, or point REC_WHISPER_MODEL at your own path.

pacman_pkgs := ffmpeg libpulse whisper-cpp

go_tools := \
	github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest \
	gotest.tools/gotestsum@latest \
	github.com/vladopajic/go-test-coverage/v2@latest \
	github.com/goreleaser/goreleaser/v2@latest

deps_runtime := ffmpeg pactl whisper-cli claude
deps_dev := golangci-lint gotestsum go-test-coverage goreleaser

whisper_model_dir := $(if $(XDG_CACHE_HOME),$(XDG_CACHE_HOME),$(HOME)/.cache)/whisper.cpp
whisper_model_name := ggml-large-v3-turbo.bin
whisper_model := $(whisper_model_dir)/$(whisper_model_name)
whisper_model_url := https://huggingface.co/ggerganov/whisper.cpp/resolve/main/$(whisper_model_name)

setup: deps-system deps-go whisper-model deps-check

deps-system:
	@echo "==> Installing system packages" >&2
	@if ! command -v pacman >/dev/null; then \
		echo "pacman not found. Install these yourself: $(pacman_pkgs)" >&2; \
		exit 1; \
	fi
	@sudo pacman -S --needed $(pacman_pkgs)

deps-go:
	@echo "==> Installing Go tools" >&2
	@for m in $(go_tools); do \
		echo "  $$m" >&2; \
		go install "$$m"; \
	done

whisper-model:
	@if [[ -f "$(whisper_model)" ]]; then \
		echo "==> Whisper model already present: $(whisper_model)" >&2; \
	else \
		echo "==> Downloading the whisper model, about 1.6GB" >&2; \
		mkdir -p "$(whisper_model_dir)"; \
		curl -fL --progress-bar -o "$(whisper_model)" "$(whisper_model_url)"; \
	fi

deps-check:
	@echo "==> Checking dependencies" >&2
	@missing=0; \
	for c in $(deps_runtime) $(deps_dev); do \
		if command -v "$$c" >/dev/null; then \
			printf "  %-18s %s\n" "$$c" "$$(command -v "$$c")"; \
		else \
			printf "  %-18s MISSING\n" "$$c"; \
			missing=1; \
		fi; \
	done; \
	if [[ -f "$(whisper_model)" ]]; then \
		printf "  %-18s %s\n" "whisper model" "$(whisper_model)"; \
	else \
		printf "  %-18s MISSING (make whisper-model)\n" "whisper model"; \
		missing=1; \
	fi; \
	exit $$missing
