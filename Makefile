.PHONY: build test run-example

build:
	go build -o bin/config-guard ./cmd/main.go

test:
	go test ./...

run-correctness: build
	./bin/config-guard scan \
		--config ./config/config.yaml \
		--application svcA \
		--mode correctness \
		--instances uat-aps1

run-diff: build
	./bin/config-guard scan \
		--config ./config/config.yaml \
		--application svcA \
		--mode diff \
		--instances uat-aps1,staging-aps1
