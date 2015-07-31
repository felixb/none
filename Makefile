.PHONY: all clean go-clean get-deps test

all: go-clean none-scheduler test

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
