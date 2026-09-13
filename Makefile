BINARY := mdo
PKG := ./...

.PHONY: build test fmt vet tidy run

build:
	go build -o $(BINARY) ./cmd/mdo

test:
	go test $(PKG)

fmt:
	gofmt -s -w .

vet:
	go vet $(PKG)

tidy:
	go mod tidy

run:
	go run ./cmd/mdo
