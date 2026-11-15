# Contributing to tRabbit

Bug reports, feature requests and pull requests are welcome. For bugs, include your
terminal, OS, RabbitMQ version and steps to reproduce. For larger changes, open an
issue first so we can agree on the scope.

## Development

Install the tools pinned in [mise.toml](mise.toml), then build and test:

```sh
mise install
mise exec -- go build -o trabbit ./cmd/trabbit
mise exec -- go test -race ./...
mise exec -- go vet ./...
```

For a local broker with sample queues:

```sh
docker compose up -d --wait
./trabbit --url 'http://localhost:15672' --user guest --pass guest
```

Regenerate mocks with the pinned Mockery v2 configuration:

```sh
mise exec -- go generate ./...
```

[Stu](cmd/stu/README.MD) can generate queue traffic for development.

## Architecture

The UI depends on a shared application service, which delegates RabbitMQ operations to a backend interface implemented
by the HTTP client. Application and domain packages have no UI dependencies.

| Layer                                 | Responsibility                                             |
|---------------------------------------|------------------------------------------------------------|
| `cmd/trabbit`                         | Startup and dependency wiring.                             |
| `internal/app`                        | Operation contracts, validation and service orchestration. |
| `internal/client`                     | RabbitMQ HTTP transport; implements `app.Backend`.         |
| `internal/domain`                     | Shared data models.                                        |
| `internal/settings`                   | Shared preference storage and persistence.                 |
| `internal/version`                    | Application version.                                       |
| `internal/tui/engine.go`              | Scene lifecycle, active dialogs and session boundaries.    |
| `internal/tui/scenes`                 | Screen state, request ownership and refresh scheduling.    |
| `internal/tui/scenes/<scene>/widgets` | Screen-specific compositions of shared UI components.      |
| `internal/tui/components`             | Abstract UI building blocks shared across screens.         |
| `internal/tui/core`                   | Shared presentation primitives and message contracts.      |

Input flows through `Engine → scene → focused widget`. Engine owns application-wide quit confirmation and its banner;
an active dialog owns other input. Parents explicitly route messages to
children and return their commands; command results return through Engine to the owning model.

Scenes own requests and selection; their `widgets` packages own screen-specific display and editing state. Shared
`components` provide abstract controls such as inputs, selectors, forms, panels and dialogs. Widgets depend on these
building blocks and `core` presentation helpers; shared components never depend on scenes or their workflows. Chart
widgets delegate plot geometry and containment to the reusable `ChartPanel` component. Queue, consumer and binding
tables share `DataTable` viewport and footer rendering. Dashboard owns refresh scheduling.
A shared settings store is injected into screens that own preferences.

Queue detail subtabs share a bordered panel and its loading presentation while the queue table remains independent.
Engine preserves the dashboard while a full-screen operation is open, pauses hidden refresh work, and routes command results to their originating
scene. The publish scene owns request lifetime; its publish widget owns draft, focus and independent column scrolling.
The publish target starts from the selected queue and is captured with the draft at submission.

Startup configures the shared logger. Application services record operation failures; the HTTP client owns request
diagnostics and connection results. Engine and scenes log navigation and message ownership without logging UI content.
Lower layers return errors for presentation rather than logging the same failure again in the UI.

## Code guidelines

- Keep business operations in `app`, HTTP transport in `client`, and presentation state in its owning UI model. Inject
  dependencies. Put screen-specific UI in its scene's `widgets` package; keep `components` limited to reusable building
  blocks.
- Change state in `Update`; run network and file I/O in commands. Capture inputs, cancel superseded work and reject
  stale results. Command batches have no ordering guarantee.
- Route dialogs through the shared message contracts and check session, request and dialog identity before applying
  results.
- Keep rendering free of state changes. Use shared theme and layout primitives, apply each coordinate offset once, and
  stop hidden animations.
- Run `go test ./...`. Cover changed behavior and ownership boundaries. Every test and subtest must call `t.Parallel()`
  and isolate mutable state.

See [AGENTS.md](AGENTS.md) for detailed UI, keyboard, layout and testing conventions.

## Demo recording

The README demo is a real RabbitMQ session recorded with
[VHS](https://github.com/charmbracelet/vhs). Install the recorder and build the app:

```sh
mise install
mise exec -- go build -o trabbit ./cmd/trabbit
mise exec -- vhs docs/demo.tape
```

Run the tape from the repository root with RabbitMQ listening on localhost:15672
and guest/guest credentials. It creates a queue named `test` and publishes one
message. Use a disposable broker with no existing `test` queue. Run Stu separately
to show live traffic when the tape filters for `stu`.

## Releases

[GoReleaser](https://goreleaser.com) packages macOS, Linux and Windows binaries for
amd64 and arm64. Check the configuration and build a local snapshot:

```sh
mise exec -- goreleaser check
mise exec -- goreleaser release --snapshot --clean
```

Git tags are the source of truth for release versions. After CI passes on `main`,
create and push a semantic version tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The tag triggers tests and GoReleaser, which embeds its version in the binaries
and publishes archives and checksums to GitHub Releases. Choose the next tag using
patch for fixes, minor for features, and major for breaking changes. There is no
release number to update in source or configuration.

The CLI and UI share `version.Current`. Outside GoReleaser, versions come from Go's
module or VCS build metadata, including commit versions and the `+dirty` suffix
when available. Builds without version metadata show `dev`. Snapshot builds embed
a distinct snapshot version and do not publish.


### Package managers

The application repository also serves as a Homebrew tap and a Scoop bucket.
`Casks/trabbit.rb` and `bucket/trabbit.json` point to published release assets.
GoReleaser regenerates both after each stable release using the workflow's
`GITHUB_TOKEN` with `contents: write`; no separate repository or token is needed.
Keep these files generated from released assets and checksums rather than editing
versions independently. Prereleases leave the stable package manifests unchanged.

Release jobs run sequentially so two releases cannot update the manifests at the
same time. Once a tag has been published, keep its history intact and make new
changes as ordinary commits.
