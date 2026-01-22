.PHONY: all fmt lint vet test build ci clean

all: ci

fmt:
	mise run fmt

lint:
	mise run lint

vet:
	mise run vet

test:
	mise run test

build:
	mise run build

ci:
	mise run ci

clean:
	rm -f cpm
	rm -rf dist/
