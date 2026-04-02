.PHONY: cli-build cli-test cli-clean claude-message-build claude-message-test claude-message-clean

CLI_DIR := cli
CLI_BINARY := agent-message
CLAUDE_MESSAGE_DIR := claude-message
CLAUDE_MESSAGE_BINARY := claude-message

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
