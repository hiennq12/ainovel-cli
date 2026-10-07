# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`ainovel-cli` is a Go CLI/TUI that writes long-form novels (200–500+ chapters) with a multi-agent LLM architecture. This is a Vietnamese fork of `github.com/voocel/ainovel-cli` (module path unchanged). The TUI, user-facing strings, and recent commit messages are in Vietnamese. Design docs in `docs/` and some code comments are in Chinese and come from upstream. Generated novel content defaults to Vietnamese (`"language": "vi"`). `"zh"` switches to the Chinese prompt and reference sets.

## Commands

```bash
go build -o ainovel-cli ./cmd/ainovel-cli      # build
./ainovel-cli                                   # TUI (first run launches setup wizard; config at config/config.json)
./ainovel-cli --headless --prompt "..."         # headless run (also --prompt-file); no args = resume
./ainovel-cli eval --cases evals/cases/smoke [--variant dir] [--repeat N] [--ci]   # prompt/quality eval harness (calls real LLMs)
docker compose build && docker compose run --rm ainovel

# What CI runs (ubuntu + windows):
test -z "$(gofmt -l .)"
GOWORK=off go vet ./...
GOWORK=off go test -buildvcs=false -count=1 ./...
GOWORK=off go test -race -buildvcs=false -count=1 ./internal/host ./internal/store ./internal/tools

# Single test
go test ./internal/flow -run TestRoute_ExhaustiveAgainstSpec -count=1
```

`NOVEL_DIR` selects which novel workspace to operate on. The default is `workspace/`, and output goes under `<dir>/output/novel/`.

## Architecture

Start with `docs/architecture.md` (canonical). Topic docs cover the engine/arbiter (`engine-arbiter.md`, `engine-rfc.md`), context compaction (`context-management.md`), prompt caching (`prompt-cache-design.md`), evals (`evaluation-system.md`), import (`import-pipeline.md`), user rules (`user-rules-runtime.md`), and the chapter advance gate (`chapter-advance-gate.md`).

Core principle: **the fact layer is deterministic and the semantic layer is autonomous.** Decisions are placed by their nature:

- **Enumerable state transitions are code.** `internal/flow` has `LoadState → Route(state) → *Instruction`, a pure function backed by exhaustive spec tests (`router_exhaustive_test.go`). The question "who runs next after chapter N" belongs here and never in a prompt.
- **Bounded semantic judgments go to the Arbiter.** `internal/arbiter` has per-scenario `Decide*` functions: plan-start (which planner), user intervention/steer triage, and failure/deadlock resolution. Facts go in, a structured decision comes out, mechanical validation backs it up, and every decision is persisted to `decisions.jsonl` so it can be replayed.
- **Open-ended creative work goes to Workers.** These are LLM loops: `architect_short`/`architect_long`, `writer`, and `editor` (built in `internal/agents/build.go`). They are autonomous within a single chapter, review, or planning task. Each has a `CheckpointDeltaGuard`, so a worker can't finish without persisting its artifact.

Runtime layering (one-way deps): `entry (tui/headless/startup) → host → agents/arbiter → tools → store → domain`. `flow` sits above `store` and below `host`. `errs/` is usable from any layer. `diag/` subscribes to host events and reads `store/` only.

- **`internal/host`**: the shell. It owns the book lock, the logger, model config, usage/budget, and steer/intervention handling. `engine.go` is the serial deterministic loop: read store → Route → precheck → run the worker via `subagent.Runner.Run`. Deadlock bound: if Route produces the same Agent+Task again, the Arbiter is consulted at 3 repeats and a hard pause happens at 5. Subpackages `imp/` (import), `sim/` (style simulation), and `exp/` (export) hold the one-off pipelines.
- **`internal/tools`**: the only interface to the fact layer. Workers never touch Store directly. Tools return facts (e.g. `arc_end`, `needs_expansion`) and never cross-agent dispatch instructions. `commit_chapter` uses a persisted `PendingCommit` saga. `novel_context` assembles the per-role context envelope, including `reference_pack` and `working_memory.*`.
- **`internal/store`**: the filesystem store. Single-file writes are atomic (temp + fsync + rename). There are three kinds of facts: Progress, Checkpoints (step-level, in `meta/checkpoints.jsonl`, used for crash recovery), and Artifacts.
- **`internal/rules`**: deterministic lint rules. `snapshot.go` `SystemDefaults()` holds the built-in mechanical baseline (banned words, duplication, English leakage, script mixing, hook distribution). `internal/userrules` normalizes users' natural-language `.ainovel/rules/*.md` files into `meta/user_rules.json`.
- **`internal/diag`**: the observability subsystem (`/diag` report, sanitized `meta/diag-export.md`). It is observe-only: it must never auto-fix, resume, or alter flow.
- **`internal/eval`**: the `ainovel-cli eval` harness (cases in `evals/cases/`).

### Hard rules (from architecture.md "四铁律")

1. Tools return facts, not scheduling instructions.
2. Routing lives in `flow.Route`; execution lives in Engine. There is no LLM coordinator; it was retired on 2026-07-12.
3. Semantic decisions go through the Arbiter and are always persisted.
4. Hard-code only provable invariants (permissions, phase, ordering, idempotency, structure). Don't replace model judgment with keyword lists, score thresholds, or rule tables unless the decision space is closed and mechanically verifiable.

UI, logs, and diag are passive projections of the event stream. `Event.Summary` is short display text and `Event.Detail` is the full diagnostic. Truncation happens only at TUI render time.

## Assets (`assets/`, embedded via `load.go`)

Read `assets/README.md` before adding content. Summary:

- `prompts/<role>.md` holds Worker, Arbiter, and one-off task prompts, with `prompts/zh/` for Chinese. Prompts must not duplicate tool JSON schemas. Envelope paths they reference (`working_memory.*`) must match `novel_context`.
- `references/` holds writing knowledge. **Dropping a file in does nothing**: it must be wired in three places: a field on `tools.References`, `loadReferences` in `load.go`, and injection in `novel_context.go` (`writerReferences`/`architectReferences`). Genre-specific material goes in `references/genres/<style>/`.
- `styles/<style>.md` is the genre style directive appended to the writer system prompt. The filename is the `config.style` value.
- `voice.md` / `voice_zh.md` are the language voice/anti-AI-tone guidelines.
- `rules/` is deprecated. Mechanical defaults belong in `internal/rules/snapshot.go`.

Deciding where new behavior goes: if it must be *guaranteed*, put it in code (tool guards, StopAfterTools, Flow Router). Table-driven routing goes in `flow/router.go`. Semantic adjudication goes in `prompts/arbiter-*.md`. A role's aesthetic standard goes in `prompts/<role>.md`. Enumerable defaults go in `SystemDefaults()`. Knowledge material goes in `references/`.
