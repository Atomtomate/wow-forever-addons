.PHONY: lint test check
lint:
	luacheck addons spec --no-cache
test:
	busted
check: lint test
