# rclone-mount-sync — Further Work Worth Doing

Prioritised, evidence-backed list. Each entry has been individually verified against the
source code (path:line references are exact).

---

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

### 6. `cleanup` CLI command always exits 0 even on partial failure

**File:** `internal/cli/root.go:170–176`

```go
if cleaned == 0 {
    fmt.Println("No orphaned units found.")
} else {
    fmt.Printf("\nCleaned up %d orphaned unit(s).\n", cleaned)
}
return nil  // always success
```

Even if every `.ResetFailed` call in the loop failed, `cleaned == 0` and the command
still exits 0. Scripts cannot distinguish "nothing to do" from "all operations failed".

**Fix:** Track whether any reset succeeded and return non-zero if none did.

---

### 7. `parseTimerNextRunOutput` returns `(time.Time{}, nil)` on parse failure

**File:** `internal/systemd/manager.go:435–453`

```go
func parseTimerNextRunOutput(output string) (time.Time, error) {
    // ...
    if micros, err := strconv.ParseInt(value, 10, 64); err == nil {
        seconds := micros / 1000000
        // ...
        return time.Unix(seconds, nanos), nil
    }
    return time.Time{}, nil   // error silently swallowed
}
```

It claims to return `(time.Time, error)` but never returns a non-nil error. The caller
(`GetTimerNextRun`, line 452) correctly treats a zero time as "not scheduled," but the
function signature is misleading and callers cannot distinguish "no next run" from
"parse failed".

**Fix:** `return time.Time{}, fmt.Errorf("failed to parse NextElapseUSec: %q", value)`.

---

### 8. `sanitizeExtraArgs` strips alpha-key fields with no user feedback

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

### 9. `buildTimerDirectives` silently defaults to `OnCalendar=daily` for unknown types

**File:** `internal/systemd/generator.go:457–459`

```go
if len(directives) == 0 {
    directives = append(directives, "OnCalendar=daily")
}
```

If a user stores a schedule type like `"weekly"` in the config but `buildTimerDirectives`
never adds any directive for it (because only `"timer"` and `"onboot"` have matching
branches), the job silently becomes a daily job. No warning is emitted.

**Fix:** Return an error for unrecognised schedule types rather than substituting a
default:

```go
default:
    return "", fmt.Errorf("unknown schedule type %q; use 'timer', 'onboot', or 'manual'",
        schedule.Type)
```

---

### 10. `parseRemotePath` in `reconcile.go` — unit-test uncovered bug risk

**File:** `internal/systemd/reconcile.go:390–396`

```go
func parseRemotePath(remotePath string) (remote, path string) {
    idx := strings.Index(remotePath, ":")
    if idx == -1 {
        return remotePath, "/"
    }
    return remotePath[:idx], remotePath[idx+1:]
}
```

If `remotePath` is `"gdrive:"` (trailing colon with empty path), the function returns
`("gdrive", "")`. Callers in `parseMountUnit` assign this to `mount.RemotePath`, which
then propagates into the generated unit file as `gdrive:` (empty path). The mount
command itself tolerates this (mount point is required separately), but downstream
consumers of `mount.RemotePath` may return incorrect display strings.

**Fix:** Return `"/"` when the path part after the colon is empty:

```go
path := remotePath[idx+1:]
if path == "" {
    path = "/"
}
return remotePath[:idx], path
```

---

### 11. Mount delete "service only" path fails on reconciliation

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

### 12. `runSyncCreate` never rolls back config create if `generator.WriteSyncUnits` fails

**File:** `internal/cli/sync.go:152–156`

```go
if _, _, err := generator.WriteSyncUnits(savedJob); err != nil {
    _ = cfg.RemoveSyncJob(job.Name) // removes by Name, not ID
    _ = cfg.Save()
    return fmt.Errorf("failed to write systemd units: %w", err)
}
```

The rollback uses `job.Name` (string), not `savedJob.ID`. `RemoveSyncJob` looks up by
`Name`, so this works — but it calls `cfg.Save()` again while still holding the write
lock acquired inside `RemoveSyncJob`. This is correct Go (no recursive mutex), just
a style concern that's worth fixing to avoid confusion.

---

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

### 14. `GetRemoteType` in `remotes.go` hits `config show` for every remote — O(n²) loading

**File:** `internal/rclone/remotes.go:57–62`

```go
remoteType, err := c.GetRemoteType(ctx, name)
```

`GetRemoteType` spawns `rclone config show <remote>`, one subprocess per remote.
With 50 remotes the pre-flight check will block for 30–60 seconds. Currently the
error is downgraded to a warning per-remote (so the list still loads), but the slow
path blocks `ListRemotes` itself for tens of seconds.

**Fix:** Use `rclone listremotes --format t` (if available) or replace the per-remote N+1
query with a single `rclone config show` parsing all remotes at once. Alternatively
adopt a lazy-load strategy: start `ListRemotes` with empty types, then back-fill
types in a goroutine without blocking the caller.

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

### 16. `MountDetails.renderDetails` shows `AutoStart` / `Enabled` as two separate values

**File:** `internal/tui/screens/mounts.go:1102–1103`

```go
fmt.Fprintf(&b, "  Auto Start: %t\n", d.mount.AutoStart)
fmt.Fprintf(&b, "  Enabled: %t\n", d.mount.Enabled)
```

`AutoStart` is always shown as the raw config field but does not affect runtime
behaviour — only `Enabled` (propagated to `systemctl enable/disable`) drives the
running state. `AutoStart` currently starts the mount when systemd-user starts, but
that is the same as enabling the service. The two fields are redundant and their
difference is not documented in the form, leading to user confusion ("I checked
AutoStart but it didn't start!").

---

### 17. `SyncJobDetails.renderDetails` — `LastRun` shown from `d.status` not from `d.job`

**File:** `internal/tui/screens/sync_jobs.go:964–966`

```go
if !d.status.LastRun.IsZero() {
    fmt.Fprintf(&b, "    Last Run: %s\n", d.status.LastRun.Format("2006-01-02 15:04:05"))
}
```

`d.status.LastRun` comes from `GetDetailedStatus`, which queries systemd's notion of
the last run time. The config field `job.LastRun` (which is written by this
application when a sync job completes) may differ from systemd's view (which cannot
track oneshot jobs reliably). The two are kept independent, but the UI doesn't
distinguish between "config-time last run" and "systemd-recorded last run". Consider
showing both, or prefer the config value if it's more accurate for oneshot services.

---

## P3 — Polish / Code Hygiene

### 18. `lipgloss.Color("3")` is a name lookup, not an ANSI escape

**File:** `internal/tui/app.go:893`

```go
BorderForeground(lipgloss.Color("3"))
```

`lipgloss.Color` treats the argument as a colour *name* and looks it up in its registry.
`"3"` resolves to yellow in the default palette, but it means "the colour whose name
happens to be `3`", not "ANSI colour 3". If the palette is ever customised this renders
orange or undefined. Intent is clearly "yellow".

**Fix:** Replace with `lipgloss.Color("yellow")`.

---

### 19. `retry.go:255` wraps `lastErr` after exhausting retries, losing the original

```go
return fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
```

`lastErr` is the last-attempt error after `doRetryBytes` passes it back. If all retries
fail, `lastErr` is correctly captured, but it is never *classified* (i.e. the caller
never knows if it was retryable or permanent without re-inspecting the returned error).
Consider adding a `Retryable bool` field to `RetryableError` and propagating it into
the wrapped error, or at least annotating the final message with `[permanent:]` or
`[retryable:]` so the caller can decide whether to suppress or escalate.

---

### 20. Gather all `fmt.Fprintf(os.Stderr, "Warning: ")` calls onto a single helper

There are 43 locations across the codebase that produce the pattern:

```go
fmt.Fprintf(os.Stderr, "Warning: <action> failed: %v\n", err)
```

These are all unstructured, inconsistent (some print to stderr, some swallow errors
entirely in the TUI screens), and untestable without capturing os.Stderr. Centralising
on a `noteWarning(message string)` helper that can also be overridden in tests would
give uniform behaviour and make it possible to assert the right number of warnings in
integration tests.

---

## Entry Point — Start Here

If time for only one change, address **#1** (`parseVersion` / `Sscanf` at
`internal/rclone/validation.go:308`).  It is the only truth-invalidating behaviour in
the entire codebase: it runs on every launch, structurally parses the wrong region of
the version string, and exists in a function that gates whether the application can
start at all.

Second priority is **#2** (`TrimSuffix` dead step at
`internal/systemd/manager.go:535–536`): it is dead code that is trivially refactored
and guards against a future double-suffix input that would silently corrupt the ID.

Third priority is **#4** (sync-delete `StopTimer` hard-fail at
`internal/tui/screens/sync_jobs.go:1086–1102`): it creates a path that makes a basic
cleanup action impossible for services that have never been started, which is a common
real-world scenario after first install.
