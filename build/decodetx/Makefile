#!/usr/bin/make -f

BUILDDIR=build

build: go.sum
ifeq ($(OS),Windows_NT)
	exit 1
else
	go build -mod=readonly $(BUILD_FLAGS) -o build/mantlemint ./sync.go
endif


build-static:
	mkdir -p $(BUILDDIR)
	docker build --tag terramoney/decodetx ./
	docker create --name temp terramoney/decodetx:latest
	docker cp temp:/usr/local/bin/decodetx $(BUILDDIR)/
	docker rm temp

install: go.sum
	go install -mod=readonly $(BUILD_FLAGS) ./