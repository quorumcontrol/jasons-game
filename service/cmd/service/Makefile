# This is important to export until we're on Go 1.13+ or packr can break
export GO111MODULE = on

FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

projectroot=../../../

docker-build: $(prepare)
	cd $(projectroot) && make prepare && docker build -f service/cmd/service/Dockerfile .

.PHONY: docker-build prepare