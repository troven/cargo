all:

install:
	go install github.com/troven/cargo/cmd/cargo

test: export FOO_BAR=kek
test:
	cargo --dry-run \
		--context App=test/app_context.json \
		--context Friends=test/friends_context.yaml \
		test/ published/

.PHONY: test
