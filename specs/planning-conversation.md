# buff-er Planning Conversation

> Recorded 2026-03-28. Full `/feature` conversation that produced `specs/buff-er.md`.

---

## Phase 1: Intent

### Q1: What problem does this solve?

**User's initial idea:** When using AI, sometimes the AI takes a long time to finish (minutes, not hours). This leads to useless waiting time. The idea is to make that time valuable by suggesting a small amount of exercise — pushups, stretching — to keep the blood flowing and help cognition.

**Challenge:** Is the problem "dead time" or "missed nudge"?

**Refined answer:** The problem isn't idle time — it's that developers don't take physical breaks during the workday, contributing to burnout and sedentary health issues. AI wait times are the *opportunity*, not the problem itself.

**Final framing:** Developers don't take physical breaks, and natural break points (AI wait times) go unrecognized and unused for physical movement.

### Q2: How will we know it worked?

**User's initial answer:** No existing metric.

**Challenge:** "No metric" means we won't know if this worked or is just noise that gets disabled.

**User proposed:** Have the hook ask "did you do it?" — but that's self-reported and has bias.

**Agreed metric:** Primary — hook retention rate over time (are people keeping it enabled after 2+ weeks?). Secondary — self-reported exercise completion rate.

### Q3: What is out of scope?

- No personalized exercise plans or progression tracking
- No health/symptom management
- No fitness tracking or workout history (but DO track simple yes/no "did they do it")
- Customization of exercises: YES, in scope
- Cumulative stats: not v1
- Health app integration: not v1

### Q4: What must NOT happen?

1. Must not block or delay the AI operation
2. Must not suggest exercises over ~15 minutes
3. Must not fire on short tasks (erodes trust and kills retention)

### Q5: Pre-mortem

**User identified:**
- People don't like to exercise → ignore/uninstall
- Suggestions too hard → mitigated by customization
- Suggest too often → people ignore

**Added by review:**
- Bad estimation accuracy (fastest path to uninstall)
- Suggestions feel robotic/repetitive
- Wrong context (meeting, coffee shop)
- Follow-up "did you do it?" becomes guilt trip

**User's honest success framing:** "It's worth shipping if people virtue-signal installing it and starring it because it looks good for me. If users also get in better shape, great."

This changed the design priority — the suggestion UX needs to be delightful, not nagging.

---

## Phase 2: Behavioral Spec

### Stories

**Today:** AI task starts → developer stares at terminal or drifts to Slack/coffee → no physical movement.

**Desired:** AI task starts → hook estimates duration → suggests exercise → developer exercises → AI finishes → "did you do it?" → back to work.

### Key Design Insight: Permission Batching

**Problem discovered:** Users often can't walk away during AI tasks because the AI keeps asking for permissions (approve tool calls). This is why people go to YOLO mode.

**User's take:** "Without [permission batching] the product fails because users are stuck or go to YOLO mode."

**Resolution:** Permission batching is a separate but related feature — architecturally decoupled, built alongside, can be extracted later.

### Mechanism

Two trigger modes that coexist:

1. **Learned command estimation** — track actual command durations per-project, suggest before commands historically taking 3+ min
2. **Time since last break** — suggest at next natural pause if user hasn't moved in 30+ minutes

The causal chain: cool concept → installs → nudge at right moment → some exercise → benefit. But primary value is marketing/visibility.

### States and Transitions

Idle → Estimating → Decision gate → Exercise suggestion → Waiting → Complete ("did you do it?") → Idle

### Error Cases

- Bad estimate: accept risk, track accuracy
- Empty exercise list: log, warn, skip
- Rapid successive tasks: may double-fire, acceptable
- Hook crash: must not break AI workflow
- Permission detection fails: needs graceful timeout

---

## Phase 3: Design Approach

### Technical Research

**Claude Code hooks investigation revealed:**
- No queue or plan visibility — hooks fire one tool at a time
- `PreToolUse` gives tool name + input (e.g., the bash command)
- `PostToolUse` gives tool name + response
- No timing data in payloads
- `systemMessage` in output shows user-facing text without affecting Claude
- Empty stdout + exit 0 = silent no-op

**Estimation approach:**
- Option A (time-based Stop hook): REJECTED — fires after wait, not before
- Option C (standalone daemon): REJECTED — monitors existing breaks, doesn't create them
- Option B (PreToolUse with heuristics): SELECTED — but evolved into learning-based

**Final approach:** Per-repo command timing database. Record actual durations, build statistics, trigger on commands with 3+ samples averaging over threshold.

**Statistical design:**
- 3 samples minimum
- Check average + 75th percentile
- 2-3 day sliding window so stale data ages out

**Storage:** `~/.local/share/buff-er/` (XDG standard), per-project keyed by path hash

### Language Decision

**Go over Rust.** User's rationale: "I'm not good at either. I'm not reading this. It's AI all the way baby." Go chosen because it's easier for AI to vet and ensure is bug-free — simpler language, fewer footguns, bugs more visible.

### Packages Verified

- `modernc.org/sqlite` — pure-Go SQLite, no CGO
- `github.com/adrg/xdg` — XDG base directories
- `github.com/pelletier/go-toml/v2` — TOML config parsing
- `github.com/spf13/cobra` — CLI framework with subcommands

---

## Phase 4: Implementation Design

### Architecture

```
buff-er/
├── cmd/buff-er/        (main.go, hook.go, install.go, doctor.go)
├── internal/hook/      (input.go, output.go — JSON structs)
├── internal/timing/    (store.go, command.go, estimator.go, pending.go)
├── internal/exercise/  (catalog.go, suggest.go)
├── internal/config/    (config.go)
```

### Key Design Decisions

- **SQLite over flat JSON** — handles concurrent sessions natively
- **Pending entries via temp files** — each hook invocation is a separate process, so PreToolUse/PostToolUse correlation needs persistence. Uses `$TMPDIR/buff-er-{session}/` with files per tool_use_id.
- **Command grouping by first 2 tokens** — `cargo build --release` → `cargo build`. Handles pipes (use last command), env vars (skip), chains (use last).
- **All failures exit 0** — surface errors via `systemMessage`, never break AI workflow
- **Install merges, doesn't overwrite** — `settings.json` hook entries appended to existing arrays

### Naming

Brainstormed: ai-fit, deskfit, standup, swolehook, pushup, flexbreak, git-fit, idle-hands, buff-er, rep, moveit, budge.

- git-fit: taken (existing GitHub project)
- swolehook: available
- buff-er: available, great pun (buffer time + getting buff)

**Selected: buff-er** — memorable, marketable, the pun works.

### Blind Spots Identified

- Concurrent SQLite writes: handled by WAL mode
- Config parse failures: fall back to defaults
- `install.sh` modifying settings.json: must merge, not overwrite — this is where bugs hide
- Go package names: AI tends to invent them — all verified to exist
- On failure: inject `systemMessage` about the issue rather than silent failure

### Rollback

Uninstall removes hook entries. Binary without hook config does nothing. Can also disable via config `enabled: false`.
