OWNER               := hitokoto
BIN_NAME            := worker
PROJECT_NAME        := notification_worker
GIT_COMMIT          := $(shell git rev-parse HEAD)
GIT_DIRTY           :=
BUILD_TAGS          :=
VERSION             := $(shell grep "Version" version.go | awk '{print $$4}' | sed 's/\"//g')
BUILDBOX_BRANCH     := $(shell echo $$BUILDBOX_BRANCH)
BUILD_TIME          := $(shell date "+%F %T")
BUILD_TIME_WINDOWS  := $(shell date +"%Y-%m-%d %T")
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
   	-ldflags "-X 'main.BuildHash=$(GIT_COMMIT)$(GIT_DIRTY)' -X 'main.BuildTime=$(2)'" \
   	-o ./bin/$(BIN_NAME)_$(VERSION)_$(1)
endef

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(call build,linux_amd64,$(BUILD_TIME))
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(call build,darwin_amd64,$(BUILD_TIME))
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(call build,windows_amd64.exe,$(BUILD_TIME))
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(call build,linux_arm64,$(BUILD_TIME))

build-pwsh:
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(call build,linux_amd64,$(BUILD_TIME_WINDOWS))
	env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(call build,darwin_amd64,$(BUILD_TIME_WINDOWS))
	env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(call build,windows_amd64.exe,$(BUILD_TIME_WINDOWS))
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(call build,linux_arm64,$(BUILD_TIME_WINDOWS))
