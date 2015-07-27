.PHONY: all clean go-clean none

all: go-clean none

clean: go-clean
	make -C none clean

go-clean:
	go clean

none:
	make -C none