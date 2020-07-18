OWNER           := hitokoto
BIN_NAME        := worker
PROJECT_NAME    := notification_worker
GIT_COMMIT      := $(shell git rev-parse HEAD)
GIT_DIRTY       :=
BUILD_TAGS      :=
VERSION         := $(shell grep "Version" version.go | awk '{print $$4}' | sed 's/\"//g')
BUILDBOX_BRANCH := $(shell echo $$BUILDBOX_BRANCH)
.PHONY: build

ifneq ($(shell git status --porcelain) , "")
    GIT_DIRTY=+CHANGES
endif

# building the master branch on ci
ifeq ($(BUILDBOX_BRANCH) , "master")
    BUILD_TAGS :=-tags release
endif

# 构建二进制文件
define build
go build -a \
   	-ldflags "-X main.GitCommit=$(GIT_COMMIT)$(GIT_DIRTY)" \
   	-o ./bin/$(BIN_NAME)_$(VERSION)_$(1)
endef

build:
	$(call build,linux_amd64)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(call build,darwin_amd64)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(call build,windows_amd64.exe)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(call build,linux_arm64)