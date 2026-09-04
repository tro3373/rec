SHELL := bash
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir := $(patsubst %/,%,$(dir $(mkfile_path)))
PATH := $(mkfile_dir)/bin:$(PATH)
.SHELLFLAGS := -eu -o pipefail -c # -c: Needed in .SHELLFLAGS. Default is -c.
.DEFAULT_GOAL := build

dotenv := $(PWD)/.env
-include $(dotenv)

export

# Feature specific targets live under .mk/
include $(mkfile_dir)/.mk/*.mk

all: clean tidy fmt lint build test

# The one thing to run after a clone. `install` is deliberately not wired to
# `setup`: setup wants sudo, re-resolves the Go tools over the network, and ends
# in a check that fails on anything it cannot install itself. That belongs to
# the first run, not to every rebuild. Recursive so the order is kept under -j.
bootstrap:
	@$(MAKE) setup
	@$(MAKE) install
