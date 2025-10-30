#!/bin/bash

set -euo pipefail

gomods=$(find . -maxdepth 4 -name 'go.mod' | grep -v tools)

for gomod in ${gomods}; do
    cd "$(dirname "${gomod}")"

    echo "=> Updating dependencies in ${gomod}"

    OTEL_DEPS=$(grep -E '(github.com/open-telemetry/opentelemetry-collector|github.com/open-telemetry/opentelemetry-collector-contrib|go.opentelemetry.io/collector)' go.mod | grep -v '^\/\/' | awk '{print $1}' | sort -u || true)

    if [ -n "${OTEL_DEPS}" ]; then
        echo "   Found OTel dependencies:"
        echo "${OTEL_DEPS}" | sed 's/^/      /'
        echo "   Updating them..."

        for pkg in ${OTEL_DEPS}; do
            echo "      go get ${pkg}@latest"
            go get "${pkg}@latest"
        done

        echo "   Running 'go mod tidy'..."
        go mod tidy
    else
        echo "   No OTel dependencies found."
    fi

    cd -
    echo ""
done
