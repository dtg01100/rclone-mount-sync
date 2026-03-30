# Agent Customization Summary

This document summarizes the AI agent customizations created for the rclone-mount-sync project.

---

## Files Created

### 1. **AGENTS.md** (Root Level)
**Purpose:** Main bootstrap instructions for AI agents

**Location:** `/var/home/dlafreniere/projects/rclone-mount-sync/AGENTS.md`

**Contains:**
- Quick start commands (build, test, run)
- Project architecture overview
- Design patterns and conventions
- Common development tasks
- Known pitfalls to avoid
- Example prompts

**When it's used:** Automatically loaded when agents start working in this workspace

---

### 2. **TUI Development Instructions**
**Purpose:** Specialized guidance for Bubble Tea TUI development

**Location:** `/var/home/dlafreniere/projects/rclone-mount-sync/internal/tui/.instructions.md`

**Contains:**
- Elm architecture pattern explanation
- Screen navigation implementation
- Styling conventions with shared components
- Form handling with `huh` library
- Common patterns (loading states, error display)
- Testing strategies for TUI code
- Keyboard navigation standards

**Scope:** Applies to all files matching `internal/tui/**/*.go`

**When it's used:** Automatically when working on TUI-related files

---

### 3. **Systemd Integration Instructions**
**Purpose:** Specialized guidance for systemd unit file generation and management

**Location:** `/var/home/dlafreniere/projects/rclone-mount-sync/internal/systemd/.instructions.md`

**Contains:**
- Unit file template patterns
- Generator usage patterns
- Service lifecycle management
- Reconciliation for orphan units
- Error handling for systemctl commands
- Testing systemd integration
- Common pitfalls (daemon reload, permissions)

**Scope:** Applies to all files matching `internal/systemd/**/*.go`

**When it's used:** Automatically when working on systemd-related files

---

### 4. **Test Generation Skill**
**Purpose:** Domain-specific knowledge for generating high-quality tests

**Location:** `/var/home/dlafreniere/projects/rclone-mount-sync/.github/test-generation-skill.md`

**Contains:**
- Core testing principles (meaningful assertions, error conditions)
- Mocking patterns for different layers
- Table-driven test templates
- Test organization guidelines
- Anti-patterns to avoid
- Quality checklist

**When it's used:** Can be invoked explicitly or when generating test code

---

## How to Use These Customizations

### Automatic Usage

VS Code Copilot will automatically apply these instructions when:
1. Working in the workspace (AGENTS.md)
2. Editing TUI files (tui/.instructions.md)
3. Editing systemd files (systemd/.instructions.md)

### Manual Invocation

You can explicitly reference these files in prompts:

```
"Following the patterns in AGENTS.md, add a new screen for..."

"Using the systemd integration instructions, implement..."

"Generate tests following the test-generation-skill.md patterns..."
```

### Example Prompts

#### For General Development
```
"Add a new CLI command to export configuration, following the patterns in AGENTS.md"
```

#### For TUI Work
```
"Create a confirmation dialog component for mount deletion
Following the TUI development instructions"
```

#### For Systemd Work
```
"Add support for network dependency conditions in unit templates
Per the systemd integration instructions"
```

#### For Test Generation
```
"Generate comprehensive tests for the new validation function
Using the test generation skill patterns"
```

---

## Integration with VS Code

### How It Works

1. **AGENTS.md** is automatically loaded by GitHub Copilot when the workspace opens
2. **.instructions.md** files are applied based on file path patterns
3. **Skills** can be invoked via chat or automatically when context matches

### Customization Hierarchy

```
AGENTS.md (global workspace context)
    ↓
.instructions.md (directory-specific)
    ↓
Skill files (on-demand or context-triggered)
```

---

## Maintenance

### When to Update

Update these files when:
- New patterns emerge in the codebase
- Common mistakes are repeated
- New architectural decisions are made
- Additional conventions are established

### How to Update

1. Edit the relevant `.md` file
2. Keep examples consistent with actual code
3. Remove outdated patterns
4. Add new pitfalls discovered during development

---

## Related Files

These customizations complement existing documentation:

- **README.md** - User-facing documentation
- **CONTRIBUTING.md** - Human contributor guidelines
- **BUGFIXES.md** - Historical bug fixes and patterns
- **plans/architecture-design.md** - Detailed architecture
- **/memories/repo/testing-best-practices.md** - Testing guidelines

---

## Next Steps

### Suggested Enhancements

Consider adding:

1. **Rclone Integration Instructions** (`internal/rclone/.instructions.md`)
   - Retry logic patterns
   - Validation check implementation
   - Remote discovery patterns

2. **Config Management Instructions** (`internal/config/.instructions.md`)
   - XDG compliance patterns
   - Thread-safe file operations
   - Config migration strategies

3. **Custom Agent Mode** for specific workflows
   - "Bug Fix Mode" - systematic debugging approach
   - "Feature Mode" - end-to-end feature implementation
   - "Refactor Mode" - safe refactoring patterns

4. **VS Code Settings** for optimal agent experience
   - Recommended extensions
   - Key bindings for common tasks
   - Debugging configuration

---

## Quick Reference

| File | Scope | Purpose |
|------|-------|---------|
| `AGENTS.md` | Global | Workspace bootstrap |
| `internal/tui/.instructions.md` | TUI files | Bubble Tea patterns |
| `internal/systemd/.instructions.md` | Systemd files | Unit generation |
| `.github/test-generation-skill.md` | Tests | Test quality patterns |

---

## Support

For questions about these customizations:
- Review the files for detailed guidance
- Check example prompts for usage patterns
- Refer to related documentation links
- Update files when new patterns emerge
