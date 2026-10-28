#!/usr/bin/env bash
set -e

echo "Installing Bae Agent CLI..."

mkdir -p bin
go build -o bin/bae ./cmd/bae

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"
cp bin/bae "$INSTALL_DIR/bae"
chmod +x "$INSTALL_DIR/bae"

echo "Binary installed to: $INSTALL_DIR/bae"

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo "Warning: $INSTALL_DIR is not in your PATH. Add it to ~/.zshrc or ~/.bashrc:"
  echo "   export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

echo ""
echo "Setup complete! You can now run:"
echo "   bae              # Launch interactive full-screen Chat TUI"
echo "   bae -p \"prompt\"  # Run one-shot command"
