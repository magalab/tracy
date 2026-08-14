.PHONY: test vet test-cozeloop build build-web run

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

build: build-web
	go build -o bin/tracy-server ./cmd/server
