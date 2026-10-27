# Upgrading OpenTelemetry

## Current upgrade (2026-10-27)

| Family | Before | After |
| --- | --- | --- |
| Collector experimental modules | v0.142.0 | v0.164.0 |
| Collector stable modules | v1.48.0 | v1.70.0 |
| Collector contrib components | v0.142.0 | v0.164.1 |
| Go API/SDK | v1.39.0 | v1.47.0 |
| Collector builder / metadata generator | v0.136.0 | v0.164.0 |

Instrumentation, exporters, protocol modules and transitive dependencies are also
updated; their release numbering differs from the Collector's. The new Collector
and SDK require Go 1.26.0 or newer. The workspace, local tool version and CI use
Go 1.26.10.

### API migration

`scraperhelper.AddScraper` was removed upstream. The Shelly Cloud, Wavin Sentio
and ZCS Azzurro factories now use `AddMetricsScraper`. The Toggl Track logs
factory still uses the supported `AddFactoryWithConfig` API. Metadata and
component lifecycle tests were regenerated with mdatagen v0.164.0, including
previously missing Shelly Cloud tests. Provider version information in the binary
now matches its dependencies.

The unused `transformprocessor` requirement disappeared during `go mod tidy`;
it was not registered in the collector before this upgrade.

## Assessment of the old updater

The old `update_otel.sh` was useful for discovering Collector requirements and
running `go get`, but it was not a complete or sufficient upgrade process:

- It excluded `tools`, so builder and mdatagen stayed on older releases.
- It missed `go.opentelemetry.io/otel`, instrumentation, bridges, auto SDK and
  protocol dependencies.
- A separate `go get` per dependency repeatedly resolved partially upgraded
  graphs instead of updating related dependencies together.
- Workspace resolution could mask outdated receiver requirements when testing.
- Whitespace-split `find` output and invocation-relative paths were fragile.
- It neither regenerated code nor checked compilation, configuration decoding,
  lifecycle behavior, race tests or deployment builds.

The revised script updates declared direct and indirect OpenTelemetry
requirements in all six modules. It updates receivers before their locally
replaced collector consumers, resolves each module's upgrades in one `go get`
with `GOWORK=off`, tidies, then synchronizes the workspace. It handles spaces in
paths, single-line requirements and Go tool declarations. `--dry-run` lists the
planned upgrades without invoking Go or writing files. Hermetic shell tests cover
these behaviors and stop-on-error handling.

**It remains a dependency updater, not an automatic API migrator.** `@latest`
uses each module's latest tagged release at execution time; stable and
experimental modules do not share version numbers. Check the release notes and
commit the resulting module files to capture the exact resolution. Failed runs
can leave earlier modules updated: use a clean branch and inspect the diff before
retrying. Unused modules present only in upstream module graphs are not forcibly
promoted to explicit project requirements.

## Upgrade and verification procedure

Install the Go version in `.tool-versions` (for example, `asdf install golang
1.26.10`), then run from the repository root:

```sh
./update_otel.sh --dry-run
./update_otel.sh
make generate
make test
make staticcheck

# Verify each module independently, without workspace version selection.
for module in collector receiver/* tools; do
  (cd "$module" && GOWORK=off go mod tidy -diff) || exit 1
done
for module in collector receiver/*; do
  (cd "$module" && GOWORK=off go test -race ./...) || exit 1
done

# Build both deployed Linux architectures without publishing an image.
(cd collector && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/collector-linux-amd64 .)
(cd collector && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /tmp/collector-linux-arm64 .)
git diff --check
```

`make test` runs the updater's shell tests and race-enabled tests for the
collector and every receiver. Test/staticcheck targets now propagate individual
module failures instead of allowing `find -exec` to hide them. CI tests collector
and workspace/tool dependency changes as well as receiver changes.

The collector regression tests check registration of all components, decode and
validate the actual `config.yaml`, cover all receivers (including multiple named
ZCS instances), exporters and extensions, and start/stop real receiver/batch/debug
pipelines. Cloud requests are delayed and external exporters are excluded from
the lifecycle smoke test. Existing marshaler tests check log and metric output.
The rendered Helm configuration was also validated with the upgraded binary
using placeholder credentials. No live cloud account, Elasticsearch ingestion
or production deployment is validated by these checks; do a staging rollout
with real credentials before promoting the image.

After generating code, inspect `git diff`, update any provider version strings in
`collector/main.go`, and rerun generation to ensure it is reproducible. Avoid
running the repository-wide formatting target solely for a dependency upgrade,
because it also rewrites unrelated pre-existing formatting.

## Upstream references

- [Collector v0.164.0 release](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.164.0)
- [Collector API changelog](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.164.0/CHANGELOG-API.md)
- [Collector contrib v0.164.1 release](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.164.1)
- [Go API/SDK v1.47.0 release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.47.0)
