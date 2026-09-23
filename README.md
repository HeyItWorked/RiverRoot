# riverroot

A small self-hosted CI server written in Go. It reads a `pipeline.yaml`, runs each step as a local process or inside a Docker container, saves every build to SQLite, and shows the results on a dashboard.

Built to learn how CI systems work underneath: process execution, timeouts, parallelism, persistence and a REST API, with as few dependencies as possible.

## What it does

- **Steps run in order** and the build stops at the first failing step. A non-zero exit fails the build; it is not treated as an error in riverroot itself.
- **Parallel steps**: consecutive steps marked `parallel: true` run together as one batch, and the next step waits for the whole batch.
- **Container steps**: a step with an `image` runs inside that Docker image, with the working directory mounted at `/workspace`.
- **Timeouts**: each step gets 5 minutes, then it's killed.
- **Persistence**: builds and their step output are stored in SQLite (`data/riverroot.db`), so they survive restarts.
- **Dashboard + REST API** served from the same binary; the HTML is embedded at compile time.

## Quick start

Needs Go 1.26+. Container steps also need Docker.

```bash
mkdir -p data                  # the SQLite file lives here
go run ./cmd/riverroot         # dashboard on http://localhost:8080
```

Trigger a build from the dashboard's `> trigger` button, or:

```bash
curl -X POST localhost:8080/builds
```

### Demo mode

```bash
DEMO=1 go run ./cmd/riverroot
```

Seeds a couple of fake builds so the dashboard has something to show, and disables the trigger endpoint (it returns 403). Meant for public deployments where nobody should be able to run commands on the host.

## pipeline.yaml

riverroot loads `pipeline.yaml` from the directory it's started in, and runs the steps from that directory.

```yaml
name: my-app
steps:
  - name: build
    command: go build ./...
  - name: lint
    command: go vet ./...
    parallel: true
  - name: unit
    command: go test ./...
    parallel: true
  - name: integration
    command: go test -tags=integration ./...
    image: golang:1.26
```

`lint` and `unit` run together; `integration` starts once both pass. Commands go through `bash -c`, so pipes and `&&` work as they would in a terminal.

## API

| Method | Path | Returns |
|--------|------|---------|
| `GET` | `/` | the dashboard |
| `GET` | `/builds` | every build, newest first |
| `GET` | `/builds/{id}` | one build with its steps, or 404 |
| `POST` | `/builds` | runs the pipeline, saves it, returns `{"id": "..."}` (403 in demo mode) |

## Layout

```
cmd/riverroot/       entrypoint: opens the store, loads pipeline.yaml, starts the server
internal/runner/     process execution: RunCommand, RunStreaming, RunInContainer
internal/pipeline/   config loading and step execution (sequential, parallel, container)
internal/store/      persistence: SQLiteStore, JSONStore, LogStore, demo seeding
internal/git/        git helpers: latest commit, changed files, polling for new commits
internal/api/        HTTP server: REST endpoints and the embedded dashboard
```

The `git` package isn't wired into the server yet. It's the groundwork for triggering builds on new commits instead of by hand.

## Development

```bash
go vet ./...
go test ./...
```

Conventions:

- Standard library first; dependencies are only what's needed (`yaml`, `sync`, `docker`, `sqlite`, `uuid`).
- `internal/` for everything that isn't the entrypoint, so nothing is importable from outside the module.
- Errors wrap with `%w` where a caller may want to inspect the cause.
- Commit messages use conventional commits: `type(scope): subject`.
