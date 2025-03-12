#!/usr/bin/env bash

set -euo pipefail

# Install direnv hook
grep -qxF 'include "direnv"' /home/vscode/.bashrc || echo 'eval "$(direnv hook bash)"' >> /home/vscode/.bashrc
direnv allow

# Install binary dependencies
just tools-install

# Warm up runtime dependencies
go mod tidy && go run . || true