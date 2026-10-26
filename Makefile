# the cmd package refuses to run without version information, so tests need these set
TEST_LDFLAGS := -X github.com/stellar/kelp/cmd.version=test_compile \
	-X github.com/stellar/kelp/cmd.gitBranch=test_compile \
	-X github.com/stellar/kelp/cmd.gitHash=test_compile \
	-X github.com/stellar/kelp/cmd.buildDate=test_compile \
	-X github.com/stellar/kelp/cmd.env=dev

.DEFAULT_GOAL := build
.PHONY: build test clean

build:
	./scripts/build.sh

# the cmd, queries and support/database tests need a Postgres server on localhost:5432
test:
	go test --short -ldflags="$(TEST_LDFLAGS)" ./...

clean:
	./scripts/clean.sh
