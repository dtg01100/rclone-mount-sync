# Plan: teatest coverage for all screens + UX changes

## Goal

Add full-PT (program-test) coverage for the TUI using
`github.com/charmbracelet/x/exp/teatest`, complementing the existing
direct-model tests. Every screen renders, every key on every list-mode
screen produces the expected transition, and every UX change from the
recent `fix(tui): bug fixes and UX polish` commit is exercised in a
real `tea.Program`.

## Approach

### 1. Add a constructor that takes injected deps (mockable)

`App` currently builds its sub-screens in `NewApp(version)` and the
services (rclone client, generator, manager) are wired up in
`initializeServices` (a `tea.Cmd` that runs in the background and posts
`AppInitializedMsg` or `AppInitError`). The existing direct-model tests
bypass this by setting `a.rclone = nil` etc. directly.

For teatest, the program runs `Init` automatically, and a screen that
calls `rclone.IsInstalled()` against a nil client panics. We need a
clean way to construct an `App` whose services are pre-wired from
constructor args, so tests can pass `&systemd.MockManager{}`, a nil
`rclone.Client`, and a temp-dir `*config.Config`.

**Add** `NewAppWithDeps(version string, deps AppDeps) *App` in
`internal/tui/app.go`:

```go
type AppDeps struct {
    Config    *config.Config
    Rclone    *rclone.Client         // may be nil
    Generator *systemd.Generator     // may be nil
    Manager   systemd.ServiceManager // may be nil
}

func NewAppWithDeps(version string, deps AppDeps) *App { ... }
```

The body mirrors `NewApp` + the *second half* of `initializeServices`
(SetServices on each screen, NewReconciler, post `AppInitializedMsg`),
but skips `config.Load()` / `rclone.NewClient()` and accepts whatever
was passed in. Both `NewApp` and `main.go` continue to use
`initializeServices` (the production path). `NewAppWithDeps` is the
test seam.

This is a non-breaking additive change; nothing in production code
touches it.

### 2. Add the dep

```
go get github.com/charmbracelet/x/exp/teatest@latest
```

This pulls in `github.com/charmbracelet/x/exp/teatest` only — no
bubbletea upgrade, no other modules. One line in `go.mod`.

### 3. Test files

New directory `internal/tui/teatest/` for PT-driven tests, separate
from the existing `internal/tui/*_test.go` direct-model tests.

| File | Covers |
|------|--------|
| `helpers_test.go` | `newTestProgram(t, deps)` returns a `*teatest.TestModel` with a fixed 100x30 terminal, `WithProgramTimeout(2*time.Second)`, and a small helper `waitFor(t, tm, substr, timeout)` that polls `tm.Output()` until the substring appears. |
| `main_menu_test.go` | Renders main menu, asserts the 4 quick-jump letters + Quit are present, `M` navigates to mounts, `q` quits. |
| `mounts_test.go` | Empty list, list with 1 mount, list with cursor navigation, hotkey keys (`a` `e` `d` `s` `x` `t` `r` `enter` `esc`) each assert a no-op screen change (since deps are nil/mocked) or, for the start/stop keys specifically, the new "start+enable / stop+disable" sequence: assert `MockManager.LastOpName` or that `MountStatusMsg` is dispatched. |
| `sync_jobs_test.go` | Same shape as mounts. Hotkey `t` (toggle timer) and `r` (run). Assert the new `SyncJobsErrorMsg` path for toggle failures. |
| `services_test.go` | List, status, log tabs. |
| `settings_test.go` | List renders, basic key navigation. |
| `help_test.go` | `?` opens help, asserts each section is present, `esc` closes. |
| `orphan_prompt_test.go` | 4 scenarios: list with 1 orphan (Enter→import), 1 orphan (Enter→c→cleanup), error-gate (the bug fix from the previous commit), `s` (skip) removes from list, dismiss-all. |
| `live_status_test.go` | Assert the 5s tick fires by reading statuses from MockManager after a controlled time advance. The new tick types from the UX commit must actually be re-armed. |
| `ux_regression_test.go` | Smoke test for the 9 UX changes: tiny height clamp, err self-clearing, success self-clearing, suggestion rendering, delete-confirm consequences block visible, file-picker help always shows `r`. |
| `golden_test.go` | Replace the existing `snapshot_test.go` for the orphan_prompt case (since I already had to update it manually). Golden file at `testdata/golden/orphan_prompt.golden`. |

### 4. CI integration

The teatest tests use a real `tea.Program` which requires a TTY. In CI
(`ubuntu-latest` on GitHub Actions), teatest allocates a PTY via
`/dev/ptmx` — this works in the default GitHub Actions runner without
extra setup. No CI changes needed; existing
`.github/workflows/ci.yaml` already runs `go test -count=1 ./...`.

The tests should NOT depend on a real systemd user session, a real
rclone binary, or a real config dir — all are injected.

### 5. Per-test runtime budget

- Most tests: 1-2 seconds (program startup + key send + output assert)
- `live_status_test.go`: 6 seconds (waits one full 5s tick)

Total: roughly 30-40 seconds for the full teatest suite. Acceptable.

## Files to add

- `internal/tui/app.go` — add `AppDeps` struct + `NewAppWithDeps` (~40 lines)
- `internal/tui/teatest/helpers_test.go`
- `internal/tui/teatest/main_menu_test.go`
- `internal/tui/teatest/mounts_test.go`
- `internal/tui/teatest/sync_jobs_test.go`
- `internal/tui/teatest/services_test.go`
- `internal/tui/teatest/settings_test.go`
- `internal/tui/teatest/help_test.go`
- `internal/tui/teatest/orphan_prompt_test.go`
- `internal/tui/teatest/live_status_test.go`
- `internal/tui/teatest/ux_regression_test.go`
- `internal/tui/teatest/golden_test.go`
- `internal/tui/teatest/testdata/golden/orphan_prompt.golden`

## Files to modify

- `go.mod`, `go.sum` — one new line each (teatest)
- `internal/tui/snapshot_test.go` — remove the `orphan_prompt` case
  (now covered by teatest golden file)

## Validation

- `go test -count=1 -race ./internal/tui/teatest/...` — must pass
- `go test -count=1 -race ./...` — must still pass
- `make fmt`, `make lint`, `go vet ./...` — all clean

## Estimated scope

~1100 lines of new test code across 11 files, ~50 lines of production
seam in `app.go`. One dep add. One commit.
