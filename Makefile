run:
	go run ./cmd/server

build:
	go build -o bin/company-service ./cmd/server

test:
	go test ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin