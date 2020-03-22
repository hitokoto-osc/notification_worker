#!/usr/bin/env bash
set -e

#
# 来源： https://github.com/wolfeidau/somproject
#
OWNER=hitokoto
BIN_NAME=worker
PROJECT_NAME=notification_worker

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

GIT_COMMIT="$(git rev-parse HEAD)"
GIT_DIRTY="$(test -n "$(git status --porcelain)" && echo "+CHANGES" || true)"
VERSION="$(grep "const Version " version.go | sed -E 's/.*"(.+)"$/\1/' )"

# remove working build
# rm -rf .gopath
if [ ! -d ".gopath" ]; then
	mkdir -p .gopath/src/source.hitokoto.cn/${OWNER}
	ln -sf ../../../.. .gopath/src/source.hitokoto.cn/${OWNER}/${PROJECT_NAME}
fi


export GOPATH="$(pwd)/.gopath"

# move the working path and build
cd .gopath/src/source.hitokoto.cn/${OWNER}/${PROJECT_NAME}
go get -d -v ./...

# building the master branch on ci
if [ "$BUILDBOX_BRANCH" = "master" ]; then
	go build -ldflags "-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY}" -tags release -o ./bin/${BIN_NAME}
else
	go build -ldflags "-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY}" -o ./bin/${BIN_NAME}
fi
