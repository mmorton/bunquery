include .makerc

.PHONY: bootstrap-local-env
bootstrap-local-env: ## Bootstrap the local environment.
	for file in $(CFGDIR)/*.sample.toml; do cp -i "$$file" "$${file/.sample/}";done
	$(MAKE) deps
	$(MAKE) -C local up
	sleep 10
	$(MAKE) -C databases/goat migrate-auto

.PHONY: deps
deps: ## Install dependencies.
	$(GO) mod download
	$(MAKE) -C apis deps
	$(YARN) install

.PHONY: contrib
contrib: ## Setup IDE helpers.
	$(MAKE) -C contrib all

.PHONY: format
format: ## Format all code.
	$(GOFUMPT) -l -w -extra .
	$(YARN) prettier -l -w .
	terraform fmt -recursive .
	$(MAKE) -C apis format
	$(MAKE) -C databases/goat format

.PHONY: start
start:
	$(CONCURRENTLY) \
		-n "GOAT,WEB,APIS,CGEN,DB"  \
		-c "green,blue,magenta,cyan,yellow" \
		"$(MAKE) -C services/goat start" \
		"$(MAKE) -C apps/web start" \
		"$(MAKE) -C apis start" \
		"$(MAKE) -C codegen start"

.PHONY: start-backend
start-backend:
	$(CONCURRENTLY) \
		-n "GOAT,APIS,CGEN,DB"  \
		-c "green,magenta,cyan,yellow" \
		"$(MAKE) -C services/goat start" \
		"$(MAKE) -C apis start" \
		"$(MAKE) -C codegen start"

.PHONY: start-web
start-web:
	$(MAKE) -C apps/web start

.PHONY: start-desktop
	$(MAKE) -C apps/desktop start

.PHONY: clean
clean:
	$(MAKE) -C databases/goat clean
	$(MAKE) -C services/goat clean
	$(MAKE) -C apps/web clean

.PHONY: test
test: ## Run all tests.
	$(MAKE) -C services/goat test
	$(MAKE) -C apps/web test

.PHONY: build
build: ## Build all packages.
	$(MAKE) -C databases/goat build
	$(MAKE) -C services/goat build
	$(MAKE) -C apps/web build

.PHONY: package
package: ## Package all packages.
	$(MAKE) -C databases/goat package
	$(MAKE) -C services/goat package
	$(MAKE) -C apps/web package

.PHONY: ship-it
ship-it: ## Ship it.
ship-it: ecr
	$(MAKE) -C databases/goat ship-it
	$(MAKE) -C services/goat ship-it
	$(MAKE) -C apps/web ship-it

.PHONY: release
release: VERSION = $(shell $(TAG_VERSION) -patch)
release: clean deps test
	VERSION=$(VERSION) $(MAKE) build package
	@echo "Releasing $(VERSION)..."
	@echo -n "$(VERSION)" > ./VERSION
	git add ./VERSION
	git commit -m "release $(VERSION)"
	git tag -a v$(VERSION) -m "release $(VERSION)"
	git push && git push --tags
	VERSION=$(VERSION) $(MAKE) ship-it
