# the cmd package refuses to run without version information, so tests need these set
TEST_LDFLAGS := -X github.com/stellar/kelp/cmd.version=test_compile \
	-X github.com/stellar/kelp/cmd.gitBranch=test_compile \
	-X github.com/stellar/kelp/cmd.gitHash=test_compile \
	-X github.com/stellar/kelp/cmd.buildDate=test_compile \
	-X github.com/stellar/kelp/cmd.env=dev

# throwaway postgres container for the tests, on a non-standard port so it doesn't clash with a postgres on the host
# durability is turned off since the data is thrown away, and the upgrade tests assert on how long migrations take
TEST_PG_IMAGE     ?= postgres:16
TEST_PG_CONTAINER ?= kelp-test-postgres
TEST_PG_PORT      ?= 9876
TEST_PG_USER      := postgres
TEST_PG_PASSWORD  := kelp-test

.DEFAULT_GOAL := build
.PHONY: build test clean

build:
	./scripts/build.sh

# the trap removes the container however the recipe exits, including when tests fail or are interrupted
test:
	@set -e; \
	trap 'docker rm -f $(TEST_PG_CONTAINER) >/dev/null 2>&1 || true' EXIT; \
	docker rm -f $(TEST_PG_CONTAINER) >/dev/null 2>&1 || true; \
	echo "starting $(TEST_PG_IMAGE) container '$(TEST_PG_CONTAINER)' on port $(TEST_PG_PORT) ..."; \
	docker run -d --name $(TEST_PG_CONTAINER) \
		-e POSTGRES_USER=$(TEST_PG_USER) -e POSTGRES_PASSWORD=$(TEST_PG_PASSWORD) \
		-p 127.0.0.1:$(TEST_PG_PORT):5432 $(TEST_PG_IMAGE) \
		-c fsync=off -c synchronous_commit=off -c full_page_writes=off >/dev/null; \
	echo "waiting for postgres to accept connections ..."; \
	for i in $$(seq 1 60); do \
		docker exec $(TEST_PG_CONTAINER) pg_isready -q -h 127.0.0.1 -U $(TEST_PG_USER) && break; \
		if [ $$i -eq 60 ]; then echo "postgres did not become ready"; docker logs $(TEST_PG_CONTAINER); exit 1; fi; \
		sleep 1; \
	done; \
	POSTGRES_HOST=127.0.0.1 POSTGRES_PORT=$(TEST_PG_PORT) \
	POSTGRES_USER=$(TEST_PG_USER) POSTGRES_PASSWORD=$(TEST_PG_PASSWORD) \
		go test --short -count=1 -ldflags="$(TEST_LDFLAGS)" ./...

clean:
	./scripts/clean.sh
