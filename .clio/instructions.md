# rclone-mount-sync - CLIO Project Instructions

**Project Methodology:** The Unbroken Method for Human-AI Collaboration

## Scope

This file defines the **HOW** of working on rclone-mount-sync: workflow,
collaboration checkpoints, error recovery, and ownership. It contains no
technical reference (commands, file paths, stack info) — that lives in
`AGENTS.md` at the repo root.

---

## The Unbroken Method

This project follows **The Unbroken Method** for human-AI collaboration. This
is the core operational framework.

**The Seven Pillars:**

1. **Continuous Context** - Never break the conversation. Maintain momentum through collaboration checkpoints.
2. **Complete Ownership** - If you find a bug, fix it. No "out of scope."
3. **Investigation First** - Read code before changing it. Never assume.
4. **Root Cause Focus** - Fix problems, not symptoms.
5. **Complete Deliverables** - No partial solutions. Finish what you start.
6. **Structured Handoffs** - Document everything for the next session.
7. **Learning from Failure** - Document mistakes to prevent repeats.

---

## Core Workflow

```
1. Read code first (investigation)
2. Use collaboration tool (get approval)
3. Make changes (implementation)
4. Test thoroughly (verify)
5. Commit with clear message (handoff)
```

---

## Project-Specific Workflow Notes

rclone-mount-sync has no special workflow deviations from the Unbroken Method
above. Use standard conventions:

- **Conventional Commits** are mandatory (see `AGENTS.md` -> Commit Format).
- **Tests are required** for new code paths in `internal/`. Run `make test`
  before committing.
- **Pre-flight checks are part of the app**, not the agent workflow — agents
  do not need to re-run rclone/systemd pre-flight checks; tests in
  `internal/rclone/validation_test.go` cover that surface.
- **Systemd/FUSE tests are filesystem-touching** — they are unit tests that
  use temp dirs and may need a Linux host. Don't gate commits on them
  running in a sandbox where they can't.
- **TUI changes** must not break `make test` (Bubble Tea programs are
  covered by `tui_test.go` and the screen-level `*_test.go` files).

---

## Session Handoff Procedures

**When ending a session, ALWAYS create handoff directory:**

```
ai-assisted/YYYYMMDD/HHMM/
├── CONTINUATION_PROMPT.md  [MANDATORY] - Next session's complete context
├── AGENT_PLAN.md           [MANDATORY] - Remaining priorities & blockers
└── NOTES.md                [OPTIONAL]  - Technical notes
```

**Format:**
- `YYYYMMDD` = Date (e.g., `20260602`)
- `HHMM` = Time in UTC (e.g., `2240` for 22:40)

### NEVER COMMIT Handoff Files

**Before every commit:**

```bash
# Verify no handoff files staged:
git status

# If ai-assisted/ appears:
git reset HEAD ai-assisted/

# Then commit only code/docs:
git add -A && git commit -m "type(scope): description"
```

**Why:** Handoff files contain internal session context and should NEVER be in
the public repository. This project already follows conventional commits and
keeps the `main` branch clean.

### CONTINUATION_PROMPT.md

**Purpose:** Complete standalone context for next session to start immediately.

**Required Sections:**

1. **What Was Accomplished** - Completed tasks, code changes, test results
2. **Current State** - Git activity, files modified, known issues
3. **What's Next** - Priority 1/2/3 tasks, dependencies, blockers
4. **Key Discoveries & Lessons** - What you learned, patterns identified
5. **Context for Next Developer** - Architecture notes, limitations
6. **Quick Reference: How to Resume** - Commands, files, starting points

**Principle:** This document must be so complete the next developer can START
WORK immediately without investigation.

### AGENT_PLAN.md

**Purpose:** Quick reference for next session's task breakdown.

**Required Sections:**

1. **Work Prioritization Matrix** - Priority, Task, Estimated Time, Status, Blocker
2. **Task Breakdown** - Status, Effort, Dependencies, What to do, Files, Success criteria
3. **Testing Requirements** - What needs testing, how to verify, regression checks
4. **Known Blockers** - What's blocking, what's needed, workarounds

---

*For universal agent behavior (checkpoints, tool-first, ownership, error recovery, etc.), see system prompt.*
*For technical reference (code style, testing, module structure, architecture), see `AGENTS.md` at repo root.*
