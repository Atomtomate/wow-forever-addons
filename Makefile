.PHONY: lint test check golint gotest rslint rstest
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
rslint:
	cd companion-app && cargo fmt --all --check && cargo clippy --workspace --all-targets --locked -- -D warnings
rstest:
	cd companion-app && cargo test -p forever-friends-core --locked && cargo test --workspace --locked
