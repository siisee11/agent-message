.PHONY: cli-build cli-test cli-clean claude-message-build claude-message-test claude-message-clean npm-auth-check publish publish-agent-message publish-codex-message publish-claude-message

CLI_DIR := cli
CLI_BINARY := agent-message
CODEX_MESSAGE_DIR := codex-message
CLAUDE_MESSAGE_DIR := claude-message
CLAUDE_MESSAGE_BINARY := claude-message
PUBLISH_RETRIES ?= 5

define npm_publish_with_retries
	attempt=1; \
	while [ $$attempt -le $(PUBLISH_RETRIES) ]; do \
		npm publish && exit 0; \
		if [ $$attempt -eq $(PUBLISH_RETRIES) ]; then \
			exit 1; \
		fi; \
		echo "npm publish failed, retrying ($$attempt/$(PUBLISH_RETRIES))..." >&2; \
		attempt=$$((attempt + 1)); \
		sleep 5; \
	done
endef

cli-build:
	cd $(CLI_DIR) && go build -o ../$(CLI_BINARY) .

cli-test:
	cd $(CLI_DIR) && go test ./...

cli-clean:
	rm -f ./$(CLI_BINARY)

claude-message-build:
	cargo build --manifest-path $(CLAUDE_MESSAGE_DIR)/Cargo.toml

claude-message-test:
	cargo test --manifest-path $(CLAUDE_MESSAGE_DIR)/Cargo.toml

claude-message-clean:
	rm -rf $(CLAUDE_MESSAGE_DIR)/target

npm-auth-check:
	npm whoami >/dev/null

publish: npm-auth-check publish-agent-message publish-codex-message publish-claude-message

publish-agent-message:
	$(npm_publish_with_retries)

publish-codex-message:
	cd $(CODEX_MESSAGE_DIR) && $(npm_publish_with_retries)

publish-claude-message:
	cd $(CLAUDE_MESSAGE_DIR) && $(npm_publish_with_retries)
