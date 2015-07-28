.PHONY: all clean go-clean scheduler executor

all: go-clean go-get scheduler executor

go-clean:
	go clean

go-get:
	go get -d -a ./*/*.go

clean: go-clean
	-rm -f none-*

scheduler: scheduler/*.go
	go build -o none-$@ scheduler/*.go

executor: executor/*.go
	go build -o none-$@ executor/*.go
