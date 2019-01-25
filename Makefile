all:

install:
	go install github.com/troven/cargo

test:
	cargo -d \
		--context App=test/app_context.json \
		--context Friends=test/friends_context.yaml \
		test/ published/

.PHONY: test
