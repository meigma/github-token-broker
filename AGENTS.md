<!-- BEGIN ai-protocol -->
# Agent Instructions

This file defines expectations for any agent working in this repository. It covers **startup**, **session intent**, **skill loading**, **branching**, and **session journaling**.

---

## Sessions are user-initiated

Sessions exist only when the user explicitly asks for one. If they say "new session" / "start a session", invoke the `session-new` skill. If they say "continue session N", invoke `session-continue`. Otherwise, **do not create a session** — most conversations are discussions, questions, or small one-shot tasks that should not be journaled.

Default to no session. Do not silently prime a `.journal/<ID>/` folder on the user's behalf. If the request is ambiguous (substantial work without explicit framing), ask before journaling.

---

## Startup (Mandatory)

Before doing **anything else** in this conversation — before priming a new journal folder, before reading code, before responding substantively to the user — you **must** read the `SUMMARY.md` of the **last three sessions** in `.journal/`. This applies to every conversation in this repo, not just ones where the user opens a session.

- Look up the three highest session IDs in `.journal/` that have a `SUMMARY.md` present. Sessions without a `SUMMARY.md` (e.g. still in progress or abandoned without close-out) are skipped — continue walking backward until you have three, or until you run out.
- If fewer than three closed sessions exist, read whatever is available. Do not complain about the shortfall.
- Read each `SUMMARY.md` in full. Do **not** read their `NOTES.md` files at startup — those are large and will blow up your context. The summaries exist precisely so you can jumpstart cheaply.

This is **non-optional**. It is how you inherit continuity across the many sessions this repo will accumulate over time. Skipping it means acting without context that has already been established.

After the three summaries are read, proceed with the user's actual request. If they asked to open a new session or continue an existing one, follow the `session-new` / `session-continue` protocol below. If they did not, just help with what they asked — no journaling.

> Exception: when a session is being **continued**, you additionally read that specific session's `NOTES.md` in full per the continuing-session rules below. The three-summary startup read still applies first.

### Index as a research resource

The top-three read covers recent context. `.journal/INDEX.md` is the full map — treat it as a search tool. When the user gives you a task, scan the index for prior sessions whose Title or Summary suggests relevant history (an earlier fix in the same area, a prior design decision, an abandoned approach, a related refactor) and read only those specific `SUMMARY.md` files.

Be selective. Do **not** read every session, and do **not** read `NOTES.md` files speculatively — they are large. The index exists so you can find the few summaries that matter without a full sweep.

---

## Skill Loading (Mandatory)

### Always loaded

These skills are **mandatory** for every conversation in this repo, regardless of whether a session is opened. Keep them fresh in your active context — re-consult them whenever you act on what they cover:

- `git` — version control operations
- `worktrunk` — worktree-based branch isolation

Both are expected to be installed user-globally at `~/.claude/skills/`. If either is missing, tell the user before proceeding with any work that touches version control.

### Task-relevant skills

Before starting any unit of substantive work, **survey the available skills** and **load the ones relevant to that work**. Skills in this repository live under `.agent/skills/` (the source) with `.claude/skills/` symlinked to it so Claude Code finds them too; they encode the conventions, references, and guardrails this repo expects you to follow for a given technology or workflow. Additional skills installed user-globally at `~/.claude/skills/` are also available.

Rules:

- Before touching code, config, or infrastructure for a technology (e.g. Go, Python, GitHub Actions, or any of the languages and tools your project uses), check whether a skill exists for it. If one does, **load it first**. Do not wing it from prior knowledge when a skill is on disk.
- Load skills *proactively*, not reactively. Loading a skill after you've already written the wrong thing defeats the purpose.
- Load more than one if the task spans multiple domains (e.g. a change that touches both a backend service and the CI workflow that ships it may need a language skill *plus* a `github-actions` skill *plus* a `git`/PR skill).
- If you're unsure whether a skill applies, err on the side of loading it — the cost is small, the cost of ignoring its guidance is a principle violation.
- If no relevant skill exists and the work is non-trivial or recurring, mention this to the user when you're done. A missing skill is a documentation gap worth closing.

This rule is **non-optional** for the same reason the startup read is: skills are how the repo's accumulated knowledge stays portable across agents and sessions. Acting without them is acting without context that already exists on disk.

---

## Branching — Worktrunk (Mandatory)

All code changes in this repo **must** be made in a dedicated Git worktree managed by [Worktrunk](https://worktrunk.dev) (`wt`). The `worktrunk` skill is always-loaded (see Skill Loading above) — consult it for command-level detail.

### Why

Work in this repo often runs in parallel. The user may have multiple agents working on different tasks at the same time, or may context-switch across days. Worktrees keep each stream of work physically isolated — no stashing, no branch juggling, no "I forgot which branch I was on."

### Rules

1. **One worktree per PR.** Every changeset that will become a pull request gets its own worktree. Do not commit unrelated changes to the same worktree/branch.
2. **Never merge locally.** Do not use `wt merge`, `wt step push`, or `git merge` to integrate a worktree back into the local default branch. Push the branch and open a PR via `gh pr create`. Merging happens on GitHub.
3. **Inspect before creating.** Run `wt list --format=json` before creating a new worktree to avoid collisions with existing session work. Do not reuse another agent's worktree unless the user explicitly hands it off.
4. **Clean up after merge.** Once a PR is merged (or abandoned), remove the worktree with `wt remove` or `wt step prune`. Do not leave stale worktrees accumulating.
5. **Name branches clearly.** Use a descriptive name that ties back to the session or task, e.g. `session-003/feature-x` or `fix/bug-y`. The branch name is ephemeral (it disappears after squash-merge), but it should be legible while it exists.

### Worktree location

All worktrees for this repo must live under `.wt/` at the repo root. This is enforced by the user `worktrunk` config:

```toml
# ~/.config/worktrunk/config.toml
worktree-path = "{{ repo_path }}/.wt/{{ branch | sanitize }}"
```

`.wt/` is gitignored. Verify with `wt config show --full` before creating any worktree — if the template does not resolve under `{{ repo_path }}/.wt/...`, `wt switch --create` will scatter worktrees outside the repo. If the config is wrong, stop and tell the user; do not silently rewrite `~/.config/worktrunk/config.toml`.

### Typical flow

```bash
wt list --format=json                          # inspect existing worktrees
wt switch --create --base main session-003/foo # create isolated worktree under .wt/session-003-foo
# ... do work, commit ...
git push -u origin HEAD                        # push the branch
gh pr create --fill                            # open a PR
# ... review, iterate ...
# after merge:
wt remove                                      # clean up the worktree
```

---

## Session Journaling

Sessions are **opt-in** — the user asks for one explicitly, as described at the top of this doc. When opened, a session becomes a unit of continuity between agents: a future agent (or a future version of me talking to one) needs to be able to look back and understand what happened, why, and where to pick up. Conversations that never open a session leave no journal trace, and that's fine — most conversations don't need one.

All session state lives under `.journal/` at the repo root.

### Layout

```
.journal/
├── INDEX.md          # Growing table of contents across all sessions
├── 001/
│   ├── SUMMARY.md    # Postmortem — written for another agent catching up
│   └── NOTES.md      # Append-only working log — written for an agent resuming
├── 002/
│   ├── SUMMARY.md
│   └── NOTES.md
└── ...
```

- Session folders use **zero-padded 3-digit incremental IDs** (`001`, `002`, …).
- `SUMMARY.md` and `NOTES.md` are **mandatory**. Any other file in a session folder is at the agent's discretion — use them freely for scratch artifacts, diagrams, transcripts, etc.

### Session Mode

When the user opens a session, they will tell you which mode:

- **Starting a new session** (e.g. "new session", "start a session"), or
- **Continuing an existing session** (they will give you the ID).

If the user says something session-related but the mode is ambiguous, **ask before priming**. Do not guess, and do not prime on your own when no session was requested.

### New Session — Priming

Only when the user has explicitly asked to start a new session:

1. Find the highest existing session ID in `.journal/` and increment by 1 (zero-padded to 3 digits). If `.journal/` does not exist, create it and start at `001`.
2. Create the new session folder: `.journal/<ID>/`.
3. Create an empty `NOTES.md` seeded only with the frontmatter block (see schema below). Do **not** create `SUMMARY.md` yet — that's written at session close.
4. Append an initial kickoff entry to `NOTES.md` capturing the user's stated goal and the current state of the world.
5. Do **not** touch `INDEX.md` yet. The INDEX row is added when the session is closed out (or updated mid-flight if the session spans multiple days).

Only after priming is complete should you begin the user's actual request.

### Continuing a Session

When the user says "continue session N":

1. Read `.journal/<N>/NOTES.md` in full, top to bottom. This is your context.
2. Read `.journal/<N>/SUMMARY.md` if it exists (a session may have been closed and is being reopened).
3. Append a new `## <timestamp> — Resume` entry to `NOTES.md` noting what you understand the current state to be and what you're about to do.
4. Proceed with the user's request.

### During a Session

- Keep `NOTES.md` updated as you work. Append entries at meaningful checkpoints: after a decision, after a blocker, after completing a unit of work, after learning something non-obvious.
- `NOTES.md` is **append-only**. Never rewrite earlier entries. If something you wrote earlier turns out to be wrong, add a new entry correcting it.
- Timestamps use the format `YYYY-MM-DD HH:MM` in the user's local time.

### Closing a Session

When the user indicates the session is ending (or you're asked to "wrap up"):

1. Write `SUMMARY.md` using the schema below.
2. Add or update the row in `.journal/INDEX.md`.
3. Confirm to the user what was recorded.

---

## File Schemas

### `.journal/INDEX.md`

```markdown
# Session Journal

| ID  | Date       | Title                          | Status      | Summary                                               |
|-----|------------|--------------------------------|-------------|-------------------------------------------------------|
| 001 | 2026-04-15 | Initial repo setup             | complete    | Scaffolded the project layout and CI pipeline.        |
| 002 | 2026-04-16 | Add feature X                  | in-progress | Implemented the API surface; UI integration pending.  |
```

- Rows ordered oldest → newest (IDs ascend top to bottom).
- **Status**: `in-progress`, `complete`, or `abandoned`.
- **Summary**: one sentence. Updated when a session transitions state.

### `.journal/<ID>/SUMMARY.md`

Written at session close. Optimized for another agent reading cold.

```markdown
---
id: 001
title: Initial repo setup
date: 2026-04-15
status: complete
repos_touched: [this-repo]
related_sessions: []
---

## Goal
What this session set out to do, in 1–3 sentences.

## Outcome
What actually happened. State plainly whether the goal was met, partially met, or abandoned.

## Key Decisions
- Decision → reason. One bullet each. Non-obvious calls only.

## Changes
- `path/to/file` — what changed and why
- Cross-repo changes listed with repo prefix, e.g. `other-repo/cmd/foo/main.go`

## Open Threads
- Anything deferred, unresolved, or intentionally left for a future session.

## References
- Links to PRs, docs, prior sessions (`.journal/000/SUMMARY.md`), external material.
```

### `.journal/<ID>/NOTES.md`

Append-only running log. Optimized for an agent resuming the session.

```markdown
---
id: 001
title: Initial repo setup
started: 2026-04-15
---

## 2026-04-15 10:20 — Kickoff
Goal for the session: <restate>.
Current state of the world: <what's already in place>.
Plan: <rough steps>.

## 2026-04-15 10:45 — First milestone
- Did <thing>.
- User asked to adjust <X>; updated approach.
- Next: <next step>.

## 2026-04-15 11:10 — Blocker
- Hit <issue>; root cause not yet known.
- Decision: defer and continue with adjacent work; revisit after lunch.
```

Rules:
- **Append-only.** Never rewrite history; correct with a new entry.
- Timestamped headings: `## YYYY-MM-DD HH:MM — short label`.
- Each entry captures what was done, what was learned, and what's next. Blockers and decisions get their own entries and are called out explicitly.
- On resume, read top-to-bottom, then add a `## <timestamp> — Resume` entry before touching anything else.

---

## Distinction — `SUMMARY.md` vs `NOTES.md`

| Aspect       | `NOTES.md`                            | `SUMMARY.md`                          |
|--------------|----------------------------------------|---------------------------------------|
| Audience     | An agent **resuming** this session    | An agent **catching up** on this session afterward |
| Shape        | Chronological, messy, exhaustive      | Structured, clean, curated            |
| Write cadence| Continuously, append-only             | Once, at session close                |
| Style        | Lab notebook                          | Postmortem                            |

## Architecture

- Use hexagonal architecture at all times. Keep business logic isolated from CLI, filesystem, network, storage, and other external adapters.

## Process

- Prefer functional testing before calling any feature complete. Unit tests are useful, but they do not prove the tool works the way the design intends.
- Take an agile approach to development. Waterfall is explicitly forbidden: underspecify when useful, prototype early, learn from the result, and refine from working behavior.
<!-- END ai-protocol -->
