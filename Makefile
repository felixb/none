.PHONY: all clean go-clean get-deps build test release tag-release next-release

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
	go test -cover ./...

release:
	@test -n "$(VERSION)"
	@echo "prepare release of NONE v$(VERSION)"
	grep -q '^## v$(VERSION)' CHANGELOG.md
	sed -e 's/## v$(VERSION).*/## v$(VERSION)/' -i CHANGELOG.md
	sed -e 's/VERSION = .*/VERSION = "$(VERSION)"/' -i scheduler/version.go
	make all

tag-release:
	@test -n "$(VERSION)"
	grep -q '^## v$(VERSION)' CHANGELOG.md
	git commit -m "prepare release NONE v$(VERSION)" CHANGELOG.md scheduler/version.go
	git tag -m "NONE v$(VERSION)" v$(VERSION)

next-release:
	@test -n "$(VERSION)"
	@echo "prepare next development cycle: NONE v$(VERSION)-snapshot"
	sed -e '2a## v$(VERSION) (unreleased)\n\n* nothing here yet\n' -i CHANGELOG.md
	sed -e 's/VERSION = .*/VERSION = "$(VERSION)-snapshot"/' -i scheduler/version.go
	git commit -m "prepare next development cycle: NONE v$(VERSION)" CHANGELOG.md scheduler/version.go
