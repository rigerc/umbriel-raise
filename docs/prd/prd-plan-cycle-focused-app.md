# Implementation Plan: Cycle Windows of the Focused Application

## Problem

`umbriel-raise --app-id APP_ID -- COMMAND` rotates through the windows of a
*named* application. There is no way to rotate through the windows of whichever
application currently owns focus. Users who keep several terminals open want a
single keybind that means "next window of this app", without binding one chord
per application and without naming a launch command that must never run.

## Architecture

Keep the single-binary CLI and add a narrowly routed `cycle` subcommand
alongside `setup`. The selection rule is a pure function over the window
snapshot, so rotation semantics are testable without a compositor.

The retry-on-stale-window loop that `activate` already implements is extracted
into a shared `focusSelected` helper parameterised by a selection function and
a fallback. `activate` keeps spawn as its fallback; `cycle` uses a no-op. The
existing activation tests assert exact IPC call sequences, so they prove the
extraction is behaviour-preserving.

## Selection Rule

1. Find the single window with `active == true` (global focus). Umbriel marks
   `focused` per workspace, so `focused` is not usable here.
2. Take its `app_id`. An empty `app_id` cannot be matched meaningfully.
3. Collect windows with exactly that `app_id`, reusing `matchingWindows`.
4. With fewer than two matches there is nothing to rotate to.
5. Otherwise defer to the existing `selectWindow`, which returns the least
   recently focused match when a match is active — the established rotation.

Steps 1, 2 and 4 report "no target", which the command treats as a silent
success. A keybind that fires while a single-window app is focused must not
print errors or change state.

## Task Slices

| Slice | Requirements | Verification |
| --- | --- | --- |
| Routing | REQ-001 | Cycle dispatch test plus existing CLI and setup suites |
| Selection | REQ-002, REQ-003, REQ-004 | Table-driven target selection over window snapshots |
| Activation | REQ-005, REQ-006 | Fake-runner IPC sequences for rotate, no-op, stale retry |
| Surface | REQ-007 | Flag parsing, usage exit codes, help stream, README/help copy |

## Interfaces and Failure Modes

- No new external interface. Read: `umbriel windows --json`. Write:
  `umbriel msg window-focus-warp:ID`.
- `cycle` never spawns. It cannot launch an application, so it needs no launch
  command and accepts no positional arguments.
- Failures: missing CLI, compositor IPC error, malformed JSON, unexpected
  positional argument, and a window closing between discovery and focus.
- A window that closes between discovery and focus triggers one refresh. If the
  refreshed snapshot has no target, the command exits `0` rather than failing —
  the user's intent (rotate within this app) is no longer applicable.

## Non-functional Constraints

- One `windows --json` call in the common path, two at most.
- No state file: rotation rides Umbriel's MRU ordering, as `activate` does.
- `--umbriel PATH` is honoured, matching the other two entry points.
- Exit codes stay aligned with the existing contract: `0` success or no-op,
  `1` Umbriel failure, `2` invalid usage.
- All existing tests continue to pass unchanged under the race detector.

## Merge Request Plan

- Branch `feat/cycle-focused-app` off `master`, matching `feat/setup-wizard`.
- One conventional commit; additive change, no behaviour change to `--app-id`
  activation or `setup`.
- Ship code, tests, README, help text, CHANGELOG entry under `Unreleased`, and
  the requirement/task/walkthrough docs this repository keeps.
- Evidence recorded in `docs/srs/srs-walkthrough.md`: focused unit runs,
  `go test -race ./...`, `go vet ./...`, `go build ./...`, and a live rotation
  against a running Umbriel session with two windows of one application.

## Requirement Trace

Focused-app rotation objective -> REQ-001..REQ-007 -> AC-001..AC-009 ->
routing, selection, activation, and surface tests.
