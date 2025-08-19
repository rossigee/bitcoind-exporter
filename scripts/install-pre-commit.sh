#!/bin/bash
set -e

echo "Installing pre-commit hooks for bitcoind-exporter..."

# Check if pre-commit is installed
if ! command -v pre-commit &> /dev/null; then
    echo "pre-commit not found. Installing..."
    if command -v pip3 &> /dev/null; then
        pip3 install pre-commit
    elif command -v pip &> /dev/null; then
        pip install pre-commit
    else
        echo "Error: pip not found. Please install Python and pip first."
        exit 1
    fi
fi

# Install the pre-commit hooks
echo "Installing pre-commit hooks..."
pre-commit install
pre-commit install --hook-type commit-msg

# Install required Go tools
echo "Installing Go tools..."
go install golang.org/x/vuln/cmd/govulncheck@latest
go install golang.org/x/tools/cmd/goimports@latest

# Run pre-commit on all files to verify setup
echo "Running pre-commit on all files to verify setup..."
pre-commit run --all-files || echo "Some hooks failed - this is normal for the first run"

echo "Pre-commit setup complete!"
echo ""
echo "Pre-commit will now run automatically on git commit."
echo "To run manually: pre-commit run --all-files"
echo "To skip hooks: git commit --no-verify"