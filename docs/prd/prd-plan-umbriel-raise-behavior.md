# Implementation Plan: Correct Umbriel Raise Behavior

## Goal

Make `umbriel-raise` reliably focus existing windows and launch commands without
changing their arguments. Umbriel reports windows in most-recently-focused
order, exposes workspace-local `focused` and seat-global `active` state, and
runs `spawn` commands through a shell. The current implementation assumes a
stable listing order, uses workspace-local focus for cycling, and forwards raw
arguments that Umbriel later joins with spaces.

## Proposed Changes

- Use the JSON `active` field to distinguish the globally active window from
  windows remembered as focused on inactive workspaces.
- Treat Umbriel's listing as MRU order: raise the first match when another app
  is active, or rotate to the least-recently-focused match when the app itself
  is active.
- Focus through `window-focus-warp:<id>`, as recommended for runtime switchers.
- POSIX-shell-quote every launch argument before passing one command string to
  `umbriel msg spawn`.
- Document the resulting focus, cycling, and argument behavior.

## Acceptance Criteria

- AC-1: An existing matching window is focused with cursor warp and no process
  is launched.
- AC-2: Repeated calls can reach all of three or more matching windows despite
  Umbriel reordering its MRU list after every focus.
- AC-3: Workspace-local focus state cannot select the wrong cycle anchor.
- AC-4: Empty arguments, whitespace, quotes, dollar signs, and shell
  metacharacters reach a launched command literally.
- AC-5: No-match launch, stale-window retry, CLI errors, and exact app-ID
  matching retain their existing behavior.

## Task Slices

| Slice | Scope | Verification |
| --- | --- | --- |
| Selection | Active/MRU-aware window selection | Focused Go regression tests |
| Focus | Use the documented warp action | Runner boundary assertions |
| Launch | Lossless POSIX shell command construction | Shell round-trip and activation tests |
| Documentation | Align README and help text | Review plus full release gate |

## Risks

- Cursor warping is a deliberate behavior change, required for a reliable
  runtime switcher under pointer-following focus policies.
- Shell quoting must handle empty strings and embedded single quotes exactly.
- Umbriel's undocumented top-level help omission for `--json` is not a blocker;
  its repository IPC documentation defines `windows --json` as the structured
  query interface.

## Verification Plan

Use `common-tdd` for focused RED-GREEN-REFACTOR loops, then run `go test -race
./...`, `go vet ./...`, and `go build ./...`. Smoke-test both an existing match
and a safe no-match command against the running installed Umbriel compositor.

## Next Workflow

implementation-readiness
