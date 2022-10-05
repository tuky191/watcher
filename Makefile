#!/usr/bin/make -f

BUILDDIR=build

build: go.sum
ifeq ($(OS),Windows_NT)
	exit 1
else
	go build -mod=readonly $(BUILD_FLAGS) -o build/rpcwatcher cmd/rpcwatcher/main.go
	go build -mod=readonly $(BUILD_FLAGS) -o build/decodetx cmd/decodetx/main.go
	go build -mod=readonly $(BUILD_FLAGS) -o build/sync cmd/sync/main.go
endif


build-static:
	mkdir -p $(BUILDDIR)
	mkdir -p $(BUILDDIR)/static
	docker buildx build --tag terramoney/starshot .
	docker create --name temp terramoney/starshot:latest
	docker cp temp:/usr/local/bin/rpcwatcher $(BUILDDIR)/static/
	docker cp temp:/usr/local/bin/decodetx $(BUILDDIR)/static/
	docker cp temp:/usr/local/bin/sync $(BUILDDIR)/static/
	docker rm temp

install: go.sum
	go install -mod=readonly $(BUILD_FLAGS) ./