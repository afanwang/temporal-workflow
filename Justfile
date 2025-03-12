tools-install TOOL="":
  #!/usr/bin/env bash

  set -euo pipefail

  # Get root of repository.
  repo_root=$(git rev-parse --show-toplevel)
  export GOBIN="${repo_root}/bin"
  cd "${repo_root}/_tools"

  # Ensure build dependencies are downloaded and ready for build phase.
  go mod tidy
  # Retrieve all tools inside "import" block in tools.go
  all_tools=$(go list -e -f "{{{{range .Imports}}{{{{.}} {{{{end}}" ./tools.go)

  if [ -z "{{TOOL}}" ]; then
    echo "No tools specified, installing everything..."
    tools=${all_tools}
  else
    # Use tools passed as arguments.
    tools=$(echo "${all_tools}" | tr ' ' '\n' | grep -E "{{TOOL}}")
  fi

  for tool in ${tools}; do
    printf "Installing \e[94m%s\e[0m...\n" "${tool}"
    go install "${tool}"
  done
