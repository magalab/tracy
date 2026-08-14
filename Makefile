.PHONY: test vet test-cozeloop build build-web web-check run

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
