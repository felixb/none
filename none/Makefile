.PHONY: all clean

all: none-scheduler none-executor

clean:
	-rm -f none-*

none-scheduler: none_scheduler.go
	go build -o $@ $<

none-executor: none_executor.go
	go build -o $@ $<