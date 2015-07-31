.PHONY: all clean go-clean get-deps build test

all: go-clean none-scheduler test

build: go-clean none-scheduler

get-deps:
	go get -t -d -v ./...

clean: go-clean
	-rm -f none-*

go-clean:
	go clean

none-scheduler: scheduler/*.go
	go build -o $@ scheduler/*.go

test:
	go test ./...
