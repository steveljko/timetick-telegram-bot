APP_NAME = timetick-telegram-bot
SRC = main.go
BUILD_DIR = bin
GO_FILES = $(wildcard *.go)

all: build

build: $(GO_FILES)
	@echo "Building the application..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) *.go

clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)

.PHONY: all build clean
