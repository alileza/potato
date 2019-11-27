project_name = potato
branch = $(shell git symbolic-ref HEAD 2>/dev/null)
version = 0.1.0
revision = $(shell git log -1 --pretty=format:"%H")
build_user = alileza
build_date = $(shell date +%FT%T%Z)
pwd = $(shell pwd)

build_dir ?= bin/

pkgs          = ./...
version_pkg= main
ldflags := "-X $(version_pkg).Version=$(version) -X $(version_pkg).Branch=$(branch) -X $(version_pkg).Revision=$(revision) -X $(version_pkg).BuildUser=$(build_user) -X $(version_pkg).BuildDate=$(build_date)"


build:
	@echo ">> building binaries"
	@go build -mod vendor -ldflags $(ldflags) -o $(build_dir)/$(project_name) .

build-all:
	@echo ">> building all binaries"
	@gox -ldflags $(ldflags) -output dist/potato.$(version)_{{.OS}}-{{.Arch}}