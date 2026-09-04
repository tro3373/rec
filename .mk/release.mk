# goreleaser wrappers.

.PHONY: gr_init gr_check gr_snap gr_snap_skip_publish gr_build

gr_init:
	@goreleaser init
gr_check:
	@goreleaser check
gr_snap:
	@goreleaser release --snapshot --clean $(OPT)
gr_snap_skip_publish:
	@OPT=--skip-publish make gr_snap
gr_build:
	@goreleaser build --snapshot --clean
