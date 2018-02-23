BINARY_NAME=ffgrep

all: test build

build:
	go build -o $(BINARY_NAME) -v

test:
	go test -v
