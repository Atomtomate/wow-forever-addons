.PHONY: lint test check golint gotest
lint:
	luacheck addons spec --no-cache
test:
	busted
golint:
	test -z "$$(gofmt -l .)"
	go vet ./...
	GOOS=windows GOARCH=amd64 go vet ./...
gotest:
	go test ./...
check: lint test golint gotest
