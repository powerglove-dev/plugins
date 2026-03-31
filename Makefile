PLUGINS := ollama opencode claude github-copilot gemini codex

.PHONY: build install test test-e2e clean $(PLUGINS)

build: $(PLUGINS:%=build-%)

build-%:
	$(MAKE) -C plugins/$* build

install: $(PLUGINS:%=install-%)

install-%:
	$(MAKE) -C plugins/$* install

test: $(PLUGINS:%=test-%)

test-%:
	$(MAKE) -C plugins/$* test

# Run all end-to-end tests.
# Prerequisites: orcai, orcai-ollama, orcai-opencode, ollama, jq, opencode
# Tests that are missing a required tool will be automatically SKIPPED.
test-e2e:
	@echo "Running e2e tests (missing tools are SKIPped automatically)..."
	bash tests/run_all.sh

clean: $(PLUGINS:%=clean-%)

clean-%:
	$(MAKE) -C plugins/$* clean
