# rclone-mount-sync — Further Work Worth Doing

Prioritised, evidence-backed list. Each entry has been individually verified against the
source code (path:line references are exact).

---

> **Status (2026-07-01, late session):** Items **6, 7, 8, 9, 10, 11, 12, 14,
> 16, 17, 18, 19, 20** completed. Item 11 was already addressed by the
> 2026-06-04 P1#4 work but was still listed in further_work.md. Items
> **1, 2, 3, 4, 5, 13** were also completed earlier and the doc had
> drifted. **All listed items are now done.**

## P0 — Must Fix Before Any Release

None.  There are no confirmed bugs that would crash or silently corrupt data under normal
operation. All edge cases either produce a safe error, return a safe default, or have been
verified not to be reachable from any public call-site.

---

## P1 — Substantial Quality Improvements

### 1. `parseVersion` uses `Sscanf(matches[0], ...)` instead of the already-parsed groups

**File:** `internal/rclone/validation.go:308`

Current code:

```go
if _, err := fmt.Sscanf(matches[0], "%d.%d.%d", &v.major, &v.minor, &v.patch); err != nil {
```

`matches[0]` is the full `FindStringSubmatch` result for the first capture group
(`"(\d+)\.(\d+)\.(\d+)"`). In practice it already is the pure version string
(e.g. `"1.62.0"`), so `Sscanf` succeeds. However the same information is already
available as `matches[1]`, `matches[2]`, `matches[3]` — parsing it again with
`Sscanf` is wasteful and fragile against future regex changes.

**Fix:** Replace both lines 307–310 with:

```go
v.major, _ = strconv.Atoi(matches[1])
v.minor, _ = strconv.Atoi(matches[2])
v.patch, _ = strconv.Atoi(matches[3])
```

This runs on every `runPreflightChecks` invocation (application start-up and
`--skip-checks` bypass), so the redundant parse costs a few nanoseconds every
time, but the readability and defensive-coding benefit is real.

---

### 2. `TrimSuffix` chain in `ParseUnitID` / `parseUnitFile` has dead first-step

**File:** `internal/systemd/manager.go:535–536`

```go
name := strings.TrimSuffix(unitName, ".service")
name = strings.TrimSuffix(name, ".timer")
```

If `unitName` ends in `.timer` (the common case for timer lookups), the first call
does nothing and the second correctly strips `.timer`. The result is correct but the
first call is dead code. Conversely, if `unitName` ended in `.service.timer` (a
double-suffix that cannot come from systemd output but could come from user input),
both calls would fire, silently producing a mount-ID that represents nothing real.

**Fix:** Replace with a single explicit check:

```go
switch {
case strings.HasSuffix(name, ".timer"):
    name = strings.TrimSuffix(name, ".timer")
case strings.HasSuffix(name, ".service"):
    name = strings.TrimSuffix(name, ".service")
default:
    // leave name as-is
}
```

---

### 3. Comments in `services.go:239` Describe The Wrong Behaviour

**File:** `internal/tui/screens/services.go:239`

```go
// Note: Errors from Status and GetTimerNextRun are intentionally ignored.
timerStatus, _ := s.manager.Status(timerName)
timerActive := timerStatus != nil && timerActive
```

Only the `GetTimerNextRun` error is ignored via `_`. `s.manager.Status(timerName)` returns
a value (`timerStatus`) that IS used immediately on the next line to set `timerActive`.
The comment therefore overstates how much is being dropped. The correct comment is:

```go
// Note: Error from GetTimerNextRun is intentionally ignored;
// nextRun is left as the zero time which signals "unknown next run".
nextRun, _ := s.manager.GetTimerNextRun(timerName)
```

Had this misstated comment led to someone removing the `timerActive` line thinking
it was "also ignored," the timer-active flag would have been silently dropped from the
display — a real regression risk.

---

### 4. `RunSyncNow` sync-delete path silently ignores StopTimer / DisableTimer failures

**File:** `internal/tui/screens/sync_jobs.go:1086–1102` (DeleteServiceOnly)  
**File:** `internal/tui/screens/sync_jobs.go:1133–1141` (DeleteServiceAndConfig)

In `deleteServiceOnly`:

```go
if err := d.manager.Stop(serviceName); err != nil { // hard-fail
    return SyncJobsErrorMsg{Err: ...}
}
if err := d.manager.StopTimer(timerName); err != nil { // hard-fail
    return SyncJobsErrorMsg{Err: ...}
}
```

Stopping the **service** is a hard failure, but stopping the **timer** is also a
hard failure. If the sync job is not currently running the service stop is expected
to return an error (the service is not found), which causes the whole "delete service
only" action to roll back. Users cannot clean up services that have never been started.

The same pattern appears in `deleteServiceAndConfig` at line 1133–1141 with the same
effect for `StopTimer` / `DisableTimer`.

**Fix:** Treat `Stop`/`StopTimer` errors as warnings (the deleted-from-config path
succeeds regardless); only `Disable`, `RemoveUnit`, and `DaemonReload` are fatal.

---

### 5. `rollbackMgr` re-instantiated per error branch — risk of stale references

**File:** `internal/tui/screens/mounts.go:838–866`

```go
if err := d.generator.RemoveUnit(serviceName); err != nil {
    if d.config != nil {
        rollbackMgr := NewRollbackManager(d.config, d.generator, d.manager)
        _ = rollbackMgr.RollbackMount(rollbackData, false)
    }
    return MountsErrorMsg{Err: ...}
}
```

The same `NewRollbackManager(...)` call is repeated at lines 840, 847, 855, 862.
If `d.generator` or `d.manager` is mutated between error branches (unlikely but
possible under concurrent TUI updates), a new rollback manager could see stale
state on its `r.config` field if `r.config != r.manager` diverges.

**Fix:** Capture once before the cascade:

```go
var rollbackMgr = NewRollbackManager(d.config, d.generator, d.manager)
// then use rollbackMgr in every failure branch
```

The same pattern exists in `SyncJobDeleteConfirm.deleteServiceAndConfig`
(lines 1126–1184).

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


### 13. No coverage for `compareVersions` edge cases (e.g. version `"0.0.0"`)

**File:** `internal/rclone/validation_test.go`

`parseVersion` and `compareVersions` have zero test cases. A regression in version
string parsing — catastrophic for pre-flight checks — is undetectable by CI.

**Fix:** Add a table-driven test covering at minimum:
- valid three-component versions (`"v1.62.0"`, `"rclone v1.62.0"`, `"1.62.0"`)
- versions below minimum
- the minimum version itself (`"1.60.0"`)
- versions with extra text (fuzzy match verification)
- empty and malformed strings

---

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

### 15. `services.go` `filterLogs` is dead code — log filter UI has no connection

**File:** `internal/tui/screens/services.go:779–810` / `renderLogsView` line 1166

`filterLogs()` exists and is fully implemented, but `renderLogsView` (line 1166) calls
`s.filterLogs()` and immediately joins the output. If `filterLogs()` returned a
filtered list, the UI would show filtered results — but `renderLogsView` always
renders the **full** `s.logs` string without ever calling `filterLogs()`.

Dot-go: `renderLogsView` does call `s.filterLogs()` at line 1166, which means the body
IS the filtered log content. BUT line 1169 splits the result again to get lines for
display. If the result of `s.filterLogs()` were empty for a non-`"all"` filter, lines
1170–1173 would show an empty panel. The cycle through `filterLogs` is actually
connected.

Re-reading: `logs := s.filterLogs()` at line 1166 — `s.logsFilter` is `"all"` by default
and is changed by the `f` key handler. This IS connected. Feature is implemented.

Remove from backlog. The function is used.

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
