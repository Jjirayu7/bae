.PHONY: all build install doctor clean run

BINARY_NAME=bae
BIN_DIR=bin
INSTALL_DIR=$(HOME)/.local/bin

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/bae
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BIN_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed successfully! Run '$(BINARY_NAME)' from anywhere."

doctor:
	@echo "Checking Bae Ecosystem..."
	@which go >/dev/null && echo "  [OK] Go: installed" || echo "  [X] Go: NOT found"
	@which rtk >/dev/null && echo "  [OK] RTK (Token Killer): installed" || echo "  [-] RTK: optional (install for token savings)"
	@which ov >/dev/null && echo "  [OK] OpenViking: installed" || echo "  [-] OpenViking: optional (persistent memory)"
	@which ccc >/dev/null && echo "  [OK] CocoIndex: installed" || echo "  [-] CocoIndex: optional (semantic code search)"
	@which graphify >/dev/null && echo "  [OK] Graphify: installed" || echo "  [-] Graphify: optional (AST knowledge graph)"
	@which agy >/dev/null && echo "  [OK] Antigravity CLI: installed (Zero-key Auth ready)" || echo "  [-] Antigravity CLI: not found (requires GEMINI_API_KEY)"

run: build
	@./$(BIN_DIR)/$(BINARY_NAME)

clean:
	@rm -rf $(BIN_DIR)
	@echo "Clean complete."
