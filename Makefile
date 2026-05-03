BINARY_NAME=cortex
GO_FILES=$(shell find . -name "*.go")

.PHONY: all build run clean test tidy install

all: build

build: $(BINARY_NAME)

$(BINARY_NAME): $(GO_FILES)
	go build -o $(BINARY_NAME) ./cmd/cortex

run: build
	./$(BINARY_NAME)

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)

tidy:
	go mod tidy

install:
	go install ./cmd/cortex
