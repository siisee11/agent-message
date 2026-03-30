.PHONY: cli-build cli-test cli-clean

CLI_DIR := cli
CLI_BINARY := agent-messenger

cli-build:
	cd $(CLI_DIR) && go build -o ../$(CLI_BINARY) .

cli-test:
	cd $(CLI_DIR) && go test ./...

cli-clean:
	rm -f ./$(CLI_BINARY)
