# Contributing to buff-er

## Architecture Invariants

These rules keep the codebase correct. Violating any of them will cause bugs.

### 1. All hook handlers must go through `safeRun`

Every `RunE` on a hook subcommand wraps its logic in `safeRun(func() (*hook.Output, error))`. This guarantees hooks never return non-zero exit codes, which would break the host AI workflow. The `isHookInvocation()` check in `main.go` is a secondary safety net, not a substitute.

### 2. Stdout is exclusively for hook JSON

Within hook handlers, `os.Stdout` must only receive JSON written by `hook.WriteOutput`. All diagnostics go to `log.Printf` (which is set to stderr in `hook.go:init`). Any stray stdout output corrupts the Claude Code protocol.

### 3. `hook.WriteOutput(nil)` is a no-op

Hook handlers return `(nil, nil)` to signal "nothing to do." This is the idiomatic early return for non-Bash tools or insufficient data.

### 4. `PendingStore.Get` is consume-once

`Get` reads the pending file and deletes it. A second `Get` for the same ID returns nil. This prevents double-counting timing data.

### 5. State keys must use the constants in `timing` package

All keys in the `state` table must use `timing.StateKeyLastSuggestion`, `timing.StateKeyLastPrune`, `timing.StateKeySessionPrefix`, `timing.StateKeyTodayStreak`, `timing.StateKeyStreakDate`, `timing.StateKeyBreakDue`, or `timing.StateKeyPendingFollowUp`. `PruneState` identifies purgeable keys by prefix — an unknown prefix will leak rows forever.

### 6. Settings writes must use `writeSettingsAtomic`

All modifications to `~/.claude/settings.json` go through temp-file-then-rename. Direct `os.WriteFile` risks corruption on crash.

### 7. SQLite must use WAL mode + busy_timeout

`OpenStore` sets `PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000`. Without these, concurrent hook invocations fail with "database is locked."

### 8. `config.Exercise` and `exercise.ConfigExercise` must have identical fields

`GetCatalog` converts between them via direct type conversion. Adding a field to one without the other causes a compile error (safe, but the coupling is invisible).

## Adding a New Hook Event

Currently requires touching 4 files:
1. `internal/hook/input.go` — add parse function
2. `cmd/buff-er/hook.go` — add cobra command + handler
3. `cmd/buff-er/hook.go:init()` — register command
4. `cmd/buff-er/install.go:hookEntries()` — add install entry

## Running Checks

```bash
go build ./cmd/buff-er
go test ./... -count=1
golangci-lint run ./...
govulncheck ./...
nilaway ./...
sloppy-joe check
```

All six must pass before committing. The pre-commit hook runs them automatically.

## Adding Dependencies

All new Go dependencies are checked by [sloppy-joe](https://github.com/brennhill/sloppy-joe) for supply chain attacks (typosquatting, hallucinated packages, new/unvetted versions). If you add a legitimate dependency that gets flagged, add it to `~/.config/sloppy-joe/config.json` in the `allowed.go` list. The CI config in `.github/workflows/ci.yml` also has an allow list that must be updated.
