.PHONY: all clean go-clean get-deps test

all: go-clean get-deps test none-scheduler

get-deps:
	go get -d -a ./*/*.go

test:
	go test ./...

go-clean:
	go clean

clean: go-clean
	-rm -f none-*

none-scheduler: scheduler/*.go
	go build -o $@ scheduler/*.go