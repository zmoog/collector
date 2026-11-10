#!/bin/bash
# Hermetic smoke tests: no network or Go installation required.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
repo="$tmp/repo with spaces"
mkdir -p "$repo/receiver/test receiver" "$repo/collector" "$repo/tools" "$tmp/bin"
cp "$root/update_otel.sh" "$repo/"
printf 'go 1.26.0\n' > "$repo/go.work"
printf 'module example.com/collector\n' > "$repo/collector/go.mod"
printf 'require go.opentelemetry.io/collector/component v1.48.0\n' > "$repo/receiver/test receiver/go.mod"
printf '%s\n' \
    'tool go.opentelemetry.io/collector/cmd/mdatagen' \
    'require (' \
    ' go.opentelemetry.io/collector/cmd/mdatagen v0.136.0 // indirect' \
    ' go.opentelemetry.io/otel v1.39.0 // indirect' \
    ' go.opentelemetry.io/contrib/bridges/otelzap v0.13.0 // indirect' \
    ' github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.142.0 // indirect' \
    ' github.com/stretchr/testify v1.11.1' \
    ')' > "$repo/tools/go.mod"
export GO_LOG="$tmp/go.log"
export PATH="$tmp/bin:$PATH"
printf '%s\n' '#!/bin/bash' \
    'printf "%s|%s|%s\n" "$PWD" "$GOWORK" "$*" >> "$GO_LOG"' \
    'if [ "${FAIL_GET:-}" = 1 ] && [ "$1" = get ]; then exit 42; fi' > "$tmp/bin/go"
chmod +x "$tmp/bin/go"

# Invocation outside the checkout, with spaces in checkout/module paths.
cd "$tmp"
bash "$repo/update_otel.sh" --dry-run > "$tmp/dry-run"
test ! -e "$GO_LOG"
grep -q 'cmd/mdatagen@latest' "$tmp/dry-run"
grep -q 'otel@latest' "$tmp/dry-run"
grep -q 'otelzap@latest' "$tmp/dry-run"
! grep -q 'testify@latest' "$tmp/dry-run"

bash "$repo/update_otel.sh" > /dev/null
# Two modules have OTel requirements; each gets one batched get and one tidy.
test "$(grep -c '|off|get ' "$GO_LOG")" -eq 2
test "$(grep -c '|off|mod tidy' "$GO_LOG")" -eq 2
test "$(wc -l < "$GO_LOG" | tr -d ' ')" -eq 5
grep -Fq "tools|off|get github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl@latest go.opentelemetry.io/collector/cmd/mdatagen@latest go.opentelemetry.io/contrib/bridges/otelzap@latest go.opentelemetry.io/otel@latest" "$GO_LOG"
grep -Fq "|$repo/go.work|work sync" "$GO_LOG"

# A failed upgrade must stop before tidy, later modules, or workspace sync.
: > "$GO_LOG"
if FAIL_GET=1 bash "$repo/update_otel.sh" > /dev/null 2>&1; then
    echo 'Expected dependency resolution failure' >&2
    exit 1
fi
test "$(wc -l < "$GO_LOG" | tr -d ' ')" -eq 1
if bash "$repo/update_otel.sh" --unknown > /dev/null 2>&1; then
    echo 'Expected invalid argument failure' >&2
    exit 1
fi
echo 'update_otel.sh tests passed'
