.PHONY: all clean go-clean

all: go-clean go-get none-scheduler none-executor

go-clean:
	go clean

go-get:
	go get -d -a ./*/*.go

clean: go-clean
	-rm -f none-*

none-scheduler: scheduler/*.go
	go build -o $@ scheduler/*.go

none-executor: executor/*.go
	go build -o $@ executor/*.go
