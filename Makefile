.PHONY: test vet test-cozeloop build build-web build-platform web-check docker-build run

test:
	go test ./...

vet:
	go vet ./...

test-cozeloop:
	cd tests/cozeloop-e2e && go test ./...

run:
	go run ./cmd/server

build-web:
	cd web && npm install --include=dev && npm run build

web-check:
	cd web && npm run check

build: build-web
	go build -o bin/tracy-server ./cmd/server

build-platform: build-web
	@test -n "$(GOOS)" && test -n "$(GOARCH)" || (echo 'GOOS and GOARCH are required' >&2; exit 1)
	mkdir -p bin
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags='-s -w' -o bin/tracy-server-$(GOOS)-$(GOARCH) ./cmd/server

docker-build:
	docker build -t tracy:local .
