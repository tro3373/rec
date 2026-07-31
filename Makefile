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
