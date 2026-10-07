DC_PROJECT=pc_orca
DC=docker compose -f ./docker-compose.yml -p $(DC_PROJECT)
NOW=$(shell date '+%Y-%m-%d %H:%M:%S')
TESTS?=./...
.PHONY: build fmt vet lint-last-commit dc-up dc-down test gen-mocks gen-proto build-example-plugins trivy-scan dc-build


gen-mocks:
	@go tool mockery

gen-proto:
	@./scripts/gen-proto.sh

build: fmt
	@VERSION="dev" COMMIT="wip-commit" DATE="$(NOW)" ./scripts/build.sh

build-example-plugins:
	@go build -o $${ORCA_PLUGINS_PATH:-$$HOME/.orca/plugins}/orca-plugin-hello ./examples/plugins/hello

build-global: fmt
	@OUTDIR=/usr/local/bin/orca VERSION="dev" COMMIT="wip-commit" DATE="$(NOW)" ./scripts/build.sh

fmt:
	@./scripts/fmt.sh

test:
	@./scripts/test.sh $(TESTS)

vet:
	@./scripts/vet.sh

lint-last-commit:
	@npx --yes --package @commitlint/cli@20 --package @commitlint/config-conventional@20 commitlint --last

dc-up:
	$(DC) up -d

dc-build:
	$(DC) build

dc-down:
	$(DC) down --remove-orphans

trivy-scan:
	$(DC) run --rm trivy fs .