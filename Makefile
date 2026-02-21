.PHONY: test test-unit test-coverage test-coverage-html generate-mocks

test: test-unit

test-unit:
	go test ./... -count=1 -race

test-coverage:
	go test ./... -count=1 -race -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

test-coverage-html:
	go test ./... -count=1 -race -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

generate-mocks:
	mockery
