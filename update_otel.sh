#!/bin/bash

# Upgrade all OpenTelemetry modules, including SDKs, instrumentation and tools.
# Review the diff and run the compatibility checks in docs/upgrading-otel.md.
set -euo pipefail

case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [--dry-run]"
        echo "Upgrade OpenTelemetry dependencies to their latest tagged releases."
        exit 0
        ;;
    --dry-run|"") ;;
    *) echo "Usage: $0 [--dry-run]" >&2; exit 2 ;;
esac
if [ "$#" -gt 1 ]; then
    echo "Usage: $0 [--dry-run]" >&2
    exit 2
fi

cd "$(dirname "$0")"
root="$PWD"
# Update receivers before the collector, which replaces them with local modules.
# Isolate module resolution so the workspace cannot hide stale requirements.
export GOWORK=off
while IFS= read -r gomod; do
    (
        cd "$(dirname "$gomod")"
        deps=()
        while IFS= read -r pkg; do
            deps+=("${pkg}@latest")
        done < <(awk '
            $1 == "require" {sub(/^[ \t]*require[ \t]+/, "")}
            $1 ~ /^(go\.opentelemetry\.io\/|github\.com\/open-telemetry\/opentelemetry-collector(-contrib)?(\/|$))/ && $2 ~ /^v[0-9]/ {print $1}
        ' go.mod | sort -u)

        if [ "${#deps[@]}" -eq 0 ]; then
            echo "=> ${gomod}: No OTel dependencies found"
            exit 0
        fi
        echo "=> ${gomod}"
        printf '   go get'; printf ' %q' "${deps[@]}"; printf '\n'
        if [ "${1:-}" != --dry-run ]; then
            # Resolve the whole OTel family together, not one module at a time.
            go get "${deps[@]}"
            go mod tidy
        fi
    )
done < <(find receiver -name go.mod -print | sort; find collector tools -name go.mod -print | sort)

if [ "${1:-}" != --dry-run ]; then
    GOWORK="$root/go.work" go work sync
    echo "Dependencies updated. Run the compatibility checks in docs/upgrading-otel.md."
fi
