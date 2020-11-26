.PHONY: build
build:
	go build -o ./run cmd/downloader/main.go

.PHONY: gosec
gosec:
	gosec ./...

.PHONY: golint
golint:
	golint ./...

.PHONY: test
test:
	go test -cover ./...
