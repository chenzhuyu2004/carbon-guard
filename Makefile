.PHONY: build test lint docker docs-check

build:
	go build

test:
	go test ./...

lint:
	go vet ./...

docker:
	docker build -t carbon-guard-action .

docs-check:
	./scripts/check-doc-links.sh
