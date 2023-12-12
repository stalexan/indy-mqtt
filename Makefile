.DEFAULT_GOAL = help

BUILD_DIR = build
GO = go
# VERSION = $(shell git describe --tags)
VERSION = "v.0.0.1"

.PHONY: build
build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "-X 'main.version=$(VERSION)'" -o $(BUILD_DIR)/indy-mqtt ./cmd/indy-mqtt.go

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# .PHONY: test
# test:
#	$(GO) test -v ./internal/generator

.PHONY:fmt
fmt:
	$(GO) fmt ./internal/...
	$(GO) fmt ./cmd/...

.PHONY:vet
vet:
	$(GO) vet ./internal/...
	$(GO) vet ./cmd/...

.PHONY:staticcheck
staticcheck:
	staticcheck ./internal/...
	staticcheck ./cmd/...

.PHONY: help
help: ## Print this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  build          Build the binaries"
	@echo "  clean          Clean the build directory"
	@echo "  fmt            Format the code"
	@echo "  staticcheck    Check the code using staticcheck"
	@echo "  test           Run the tests"
	@echo "  vet            Check the code using vet"
	@echo "  help           Print this help message"

