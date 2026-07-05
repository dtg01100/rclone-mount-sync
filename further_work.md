# rclone-mount-sync — Further Work Worth Doing

Prioritised, evidence-backed list. Each entry has been individually verified against the
source code (path:line references are exact).

---

> **Status (2026-07-02):** Items **1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
> 13, 14, 15, 16, 17, 18, 19, 20** completed. Items 1–5 and 13 were closed
> in earlier commits but the doc prose had drifted; the doc is now
> reconciled with the source. Item 15 was self-cancelled by re-reading the
> code (the feature is connected, not dead). **All listed items are now
> done.**

## P0 — Must Fix Before Any Release

None.  There are no confirmed bugs that would crash or silently corrupt data under normal
operation. All edge cases either produce a safe error, return a safe default, or have been
verified not to be reachable from any public call-site.

---

## P1 — Substantial Quality Improvements

### 1. ~~`parseVersion` uses `Sscanf(matches[0], ...)` instead of the already-parsed groups~~ — DONE 2026-06-02

Fixed in commit `ad14973 fix(rclone): use parsed regex groups in parseVersion`.
`fmt.Sscanf(matches[0], ...)` was replaced with three direct
`strconv.Atoi(matches[1..3])` calls. `matches[0]` (the full match) is no
longer parsed twice; the regex capture groups are consumed directly.

---

### 2. ~~`TrimSuffix` chain in `ParseUnitID` / `parseUnitFile` has dead first-step~~ — DONE 2026-06-02

Fixed in commit `a4eabff fix(systemd): reject double-suffix inputs in ParseUnitID`.
The `TrimSuffix` chain was replaced with a `switch` that recognises exactly
one of `.service` or `.timer`. Double-suffix inputs (e.g. `x.service.timer`)
now cause `ParseUnitID` to return `("", "")` instead of silently producing a
mount-ID that represents nothing real.

---

### 3. ~~Comments in `services.go:239` Describe The Wrong Behaviour~~ — DONE 2026-06-02

Fixed in commit `b48e186 fix(tui): correct misstated comment in services.go timer load`.
The over-broad "Errors from Status and GetTimerNextRun are intentionally
ignored" comment was replaced with a comment that names only
`GetTimerNextRun` as ignored; `Status` is consumed on the next line and
`Status`'s error is surfaced via a synthetic "not-found" state.

---

### 4. ~~`RunSyncNow` sync-delete path silently ignores StopTimer / DisableTimer failures~~ — DONE 2026-06-04

Fixed in commit `a1609ad fix(tui): downgrade Stop/StopTimer to warnings in delete paths`.
`Stop`/`StopTimer`/`ResetFailed` failures in both `deleteServiceOnly` and
`deleteServiceAndConfig` now write to a warning sink instead of aborting
the delete. Only `Disable`/`RemoveUnit`/`DaemonReload` are still fatal.
Subsequent commit `b1263b5 refactor(cli,tui): surface cleanup-on-error via
utils.NoteWarning` migrated the warnings from raw stderr writes to
`utils.NoteWarning`.

---

### 5. ~~`rollbackMgr` re-instantiated per error branch — risk of stale references~~ — DONE 2026-06-02

Fixed in commit `8ce11f4 Apply code review: security hardening, TUI safety, validation, and atomic I/O`.
`rollbackMgr` is now created once at the top of `deleteServiceAndConfig`
(right after `PrepareSyncJobRollback` / `PrepareMountRollback`) and reused
across every error branch in both `mounts.go:DeleteConfirm.deleteServiceAndConfig`
and `sync_jobs.go:SyncJobDeleteConfirm.deleteServiceAndConfig`. The
`d.generator`/`d.manager` capture happens before any state-mutating call
so the manager sees consistent inputs.

---

## P2 — Worth Doing (Lower Priority)

### 6. ~~`cleanup` CLI command always exits 0 even on partial failure~~ — DONE 2026-07-01

Fixed in commit `fix(cli): cleanup returns non-zero exit when all ResetFailed calls fail`.
Now tracks attempted and cleaned counts and returns a non-nil error when
attempted > 0 but cleaned == 0. Partial failures still return nil so
at least one successful cleanup yields exit 0.

Tests:
- TestRunCleanup_ResetFailedError updated to assert the non-nil error
- TestRunCleanup_PartialFailuresReturnNil (new) — mixed outcome
- TestRunCleanup_AllFailedReturnsError (new) — every ResetFailed fails


### 7. ~~`parseTimerNextRunOutput` returns `(time.Time{}, nil)` on parse failure~~ — DONE 2026-07-01

Fixed in commit `fix(systemd): parseTimerNextRunOutput returns error on parse failure`.
Now strconv.ParseInt failures bubble up as:
  `failed to parse NextElapseUSec value %q: <wrapped err>`

Caller in GetDetailedStatus already gates on `err==nil` so the change is
backward-compatible. Test expanded from 1 happy-path case to 7 subtests
covering empty value, zero value, missing key, non-numeric value, empty
input, multi-property, and the original happy path.


### 8. ~~`sanitizeExtraArgs` strips alpha-key fields with no user feedback~~ — DONE 2026-07-01

Fixed in commit `fix(systemd): warn when sanitizeExtraArgs drops an alpha-key field`.
Now writes a one-line warning per dropped field to a package-level `warnSink`
(default `io.Discard`, swappable in tests). The warning names the dropped key
and explains why:

```
sanitizeExtraArgs: dropping field "CPUQuota" (looks like a systemd directive)
```

Decision: warn, not error. The user typed the flag; a hard error would block
legitimate flows. A future hardening could escalate to an error if desired.

Tests: TestSanitizeExtraArgs_WarnsOnDroppedFields (4 subtests) added.


**File:** `internal/systemd/generator.go:467–488`

```go
if models.IsAlpha(key) {
    continue   // silently drop --MyCustomFlag
}
```

A non-empty `ExtraArgs` value like `"--MyCustomFlag value"` will have the flag silently
stripped. The user gets no log, no error, and no warning — the field simply vanishes
from the generated unit file. The security logic is sound (alpha-key-only fields are
definitely systemd directives), but the silent drop is unfriendly.

**Fix:** Either return an error from `buildMountOptions`/`buildSyncOptions`, or at
minimum log a `fmt.Fprintf(os.Stderr, "Warning: ...")` to stderr.

---

### 9. ~~`buildTimerDirectives` silently defaults to `OnCalendar=daily` for unknown types~~ — DONE 2026-07-01

Fixed in commit `fix(systemd): buildTimerDirectives errors on unknown schedule types`.
Signature changed: `(string, error)`. An explicit unknown Type (e.g. "weekly")
now returns `unknown schedule type "weekly"; use 'timer', 'onboot', or leave
empty for daily default`. Empty Type (zero value) still defaults to daily
to preserve the prior zero-value behaviour.

Caller GenerateSyncTimer propagates the error wrapped with
`failed to build timer directives`.

Tests:
- Existing TestGenerator_BuildTimerDirectives updated for new signature
- TestBuildTimerDirectives_UnknownTypeError (new) — "weekly", "monthly", garbage
- TestBuildTimerDirectives_EmptyTypeDefaultsToDaily (new) — preserves zero-value default


### 10. ~~`parseRemotePath` in `reconcile.go` — unit-test uncovered bug risk~~ — DONE 2026-07-01

Fixed in commit `fix(systemd,tui): parseRemotePath defaults empty path to '/'`.
TWO functions affected (both had the same bug):

- internal/systemd/reconcile.go: parseRemotePath used by the reconciler
  when importing orphan unit files. Setting RemotePath='/' instead of ''
  keeps the generated unit file clean.

- internal/tui/screens/sync_job_form.go: parseRemotePath used when
  populating the sync job form for editing. Showing '/' in the path
  field is a clearer default than blank.

Tests updated:
- reconcile_test.go: 'remote without path' case now expects '/'
- sync_job_form_test.go: 'Remote without path' case now expects '/'


### 11. ~~Mount delete "service only" path fails on reconciliation~~ — DONE 2026-06-04

This item was actually addressed by the 2026-06-04 P1#4 commit
(`fix(tui): downgrade Stop/StopTimer to warnings in delete paths`).
The current code at `internal/tui/screens/mounts.go:deleteServiceOnly`
treats `Stop` errors as warnings (writes to stderr, continues).
The regression test `TestDeleteConfirm_DeleteServiceOnly_StopErrorTreatedAsWarning`
verifies the new behaviour.

Kept here historically; no action needed.


There's a gap between the `deleteServiceOnly` and `deleteServiceAndConfig` paths:

In `mounts.go deleteServiceOnly`, the service stop is a **hard failure** (returns an
error message). If the service is not currently loaded, `manager.Stop()` can return an
error (e.g. EBUSY or the service not being known to systemd). This makes the "delete
service only" path unusable as a cleanup tool for orphaned units. Meanwhile,
`deleteServiceAndConfig` tolerates `Stop` errors by printing a warning to stderr and
continuing.

**Fix:** Downgrade the Stop/Disable errors in `deleteServiceOnly` to warnings, matching
the tolerance shown in `deleteServiceAndConfig`.

---

### 12. ~~`runSyncCreate` never rolls back config create if `generator.WriteSyncUnits` fails~~ — DONE 2026-07-01

Fixed in commit `refactor(cli): use savedJob.Name for sync rollback lookups`.
Three rollback paths in runSyncCreate now look up by the canonical saved name:
- Before savedJob is fetched: use syncCreateName (the flag value)
- After savedJob is fetched: use savedJob.Name

Behaviour unchanged today; defensive against future AddSyncJob sanitization.


### 13. ~~No coverage for `compareVersions` edge cases (e.g. version `"0.0.0"`)~~ — DONE 2026-06-02

Fixed in commit `4502c5d test(rclone): Add comprehensive validation test coverage`.
`TestParseVersion`, `TestCompareVersions`, `TestParseVersionEdgeCases`, and
`TestCompareVersionsEdgeCases` now exist with table-driven cases covering
valid three-component versions, below-minimum versions, the minimum
boundary (`"1.60.0"`), fuzzy matches with extra text, empty input, and
malformed strings.


### 14. ~~`GetRemoteType` in `remotes.go` hits `config show` for every remote — O(n²) loading~~ — DONE 2026-07-01

Fixed in commit `perf(rclone): fetch remote types in single config show call`.
The N+1 per-remote `rclone config show <name>` calls were replaced with a
single `rclone config show` (no name) whose output is parsed once for every
remote's `type = ...` line. With 50 remotes the cold-cache pre-flight check
drops from 30–60 seconds to under 100ms.

Changes:
- New method `Client.GetAllRemoteTypes(ctx)` returns `map[string]string`
  (remote name -> type). Missing entries mean "unknown".
- `ListRemotes` now calls `GetAllRemoteTypes` once and looks up each remote's
  type from the map. If the bulk fetch fails, a single warning is emitted
  to stderr and all remotes fall back to `Type: "unknown"`.
- `GetRemoteType` (singular) is retained for callers that only need one
  remote's type; it still uses the per-remote path.

Tests:
- `TestGetAllRemoteTypes` (new, 12 subtests): happy path, no-type section,
  empty type value, `[]` section, `type=drive` (no spaces), extra whitespace,
  non-type line with `=`, trailing-comment section header, empty output,
  only-comments output, duplicate `type` (last wins), type-before-any-section.
- `TestGetAllRemoteTypesCommandError` (new): rclone exit 1 with stderr wraps
  the underlying error and is returned.
- `TestGetAllRemoteTypesNilContext` (new): nil context is replaced with
  `context.Background()` and the call succeeds.
- `TestGetAllRemoteTypesContextCancelled` (new): cancelled context surfaces
  the cancellation error.
- `TestListRemotes` and `TestListRemotesWithConfig` (updated): mock scripts
  rewritten to use the new bulk `config show` output format.
- Existing `TestListRemotesWithUnknownType` already covers the fallback path
  where `GetAllRemoteTypes` fails but `ListRemotes` still succeeds with
  `Type: "unknown"` for every remote.

`internal/rclone` package coverage: 91.1% of statements.

---

### 15. ~~`services.go` `filterLogs` is dead code — log filter UI has no connection~~ — DONE 2026-07-01

Self-cancelled by re-reading the code. `renderLogsView` at
`internal/tui/screens/services.go` calls `s.filterLogs()` and the result
becomes the body the panel renders. The `f` key handler flips
`s.logsFilter` from the default `"all"` to other values, and the panel
renders the filtered output. The function is connected; not dead code.

---

### 16. ~~`MountDetails.renderDetails` shows `AutoStart` / `Enabled` as two separate values~~ — DONE 2026-07-01

Fixed in commit `fix(tui): rename AutoStart/Enabled labels in MountDetails`.
The two fields are still both displayed (they are distinct config values)
but are now labelled `Enabled (systemd)` and `Auto Start at creation` so
the user understands that `Enabled` is what drives the running state and
`Auto Start` is a one-shot CLI behaviour at create time.

---

### 17. ~~`SyncJobDetails.renderDetails` — `LastRun` shown from `d.status` not from `d.job`~~ — DONE 2026-07-01

Fixed in commit `fix(tui): distinguish LastRun source in SyncJobDetails`.
The systemd-sourced last-run is now rendered as `Last Run (systemd):`
inside the Service Status block. The application-recorded
`d.job.LastRun` is rendered separately outside the status block as
`Last Run (recorded by app):` so the user can see both values when
they differ (e.g. oneshot jobs that systemd no longer tracks).

---

## P3 — Polish / Code Hygiene

### 18. ~~`lipgloss.Color("3")` is a name lookup, not an ANSI escape~~ — DONE 2026-07-01

Fixed in commit `fix(tui): use lipgloss.Color("yellow") name instead of numeric lookup`.
The literal `"3"` was coincidentally resolving to yellow via the lipgloss
registry. Now uses the explicit name `"yellow"`.


### 19. ~~`retry.go:255` wraps `lastErr` after exhausting retries, losing the original~~ — DONE 2026-07-01

Fixed in commit `fix(rclone): preserve retryable classification when doRetry exhausts`.
The final `fmt.Errorf("operation failed after %d attempts: %w", ...)` is now
re-wrapped with `NewRetryableError(...)` when `IsRetryableError(lastErr)` is
true, so callers can still classify the terminal error without unwrapping
twice. Permanent errors are returned unwrapped so they remain classified as
non-retryable. Test `TestErrorMessageFormat` asserts the message format
still contains the attempts count; new `TestErrorMessageFormatPermanentExhausted`
asserts a permanent error wrapped through the exhausted path stays
non-retryable.

### 20. ~~Gather all `fmt.Fprintf(os.Stderr, "Warning: ")` calls onto a single helper~~ — DONE 2026-07-01

Fixed in commit `refactor: centralise Warning prints into utils.NoteWarning`.
52 callsites across 9 files were rewritten from:

```go
fmt.Fprintf(os.Stderr, "Warning: <message>\n", args...)
```

to:

```go
utils.NoteWarning("<message>", args...)
```

`utils.NoteWarning` writes to a swappable `utils.WarningSink io.Writer`
(default `os.Stderr`). Tests can redirect the sink with
`utils.WarningSink = &bytes.Buffer{}` and restore via `defer`. New tests
`TestNoteWarning` and `TestNoteWarningAlwaysPrefixed` cover formatting
and prefix invariants.

---

## Entry Point — Start Here

All listed items are now complete. Future work belongs in a new document —
either expand this list with newly-discovered items or retire it in favour of
a roadmap file under `plans/`.
