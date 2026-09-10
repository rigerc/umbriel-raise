# Task: Cycle Windows of the Focused Application

## Requirements

- REQ-001: Route `cycle` without changing legacy activation or `setup`.
- REQ-002: Identify the focused application from global `active` state, not
  workspace-local `focused` state.
- REQ-003: Treat a missing active window or an empty `app_id` as no target.
- REQ-004: Treat a focused application with a single window as no target.
- REQ-005: Rotate to the least recently focused match, warping the cursor.
- REQ-006: Retry once when the selected window closes before focus, and exit
  successfully when the refreshed snapshot has no target.
- REQ-007: Accept `--umbriel PATH`, reject positional arguments, and document
  the subcommand in help and README.

## Acceptance Criteria

- AC-001: `cycle` dispatches and legacy `--app-id` invocations still work.
- AC-002: Workspace-local `focused` never selects the app to rotate.
- AC-003: No active window is a silent success.
- AC-004: An active window with an empty `app_id` is a silent success.
- AC-005: A single-window focused app issues no focus call.
- AC-006: Three windows of the focused app rotate to the least recent match.
- AC-007: Windows of other applications never enter the rotation.
- AC-008: A stale window ID triggers exactly one refresh and retry.
- AC-009: Usage errors exit `2`, Umbriel failures exit `1`, help exits `0`.

## Checklist

- [x] Extract the retry loop shared by `activate` and `cycle`.
- [x] Add pure focused-app target selection.
- [x] Add the `cycle` subcommand, flag set, and usage text.
- [x] Route `cycle` in the dispatcher.
- [x] Cover selection, activation, routing, and surface contracts with tests.
- [x] Update README, top-level help, and CHANGELOG.
- [x] Record verification evidence.

## Test Intent Records

| Contract | Distinct fault | Layer | Focused command |
| --- | --- | --- | --- |
| `cycle` dispatches while legacy and `setup` args do not | Subcommand is parsed as a missing `--app-id`, or `setup` routing regresses | Unit, CLI routing | `go test -run 'TestExecuteRoutesCycle$' ./...` |
| Focused-app target selection is pure and MRU-correct | Workspace-local focus picks the app, other apps leak into the rotation, or a single window self-focuses | Unit, pure selection | `go test -run 'TestSelectCycleTarget' ./...` |
| Rotation issues exactly the expected IPC calls | A no-op still warps the cursor, or rotation focuses the wrong window | Unit, fake runner | `go test -run 'TestCycleFocused' ./...` |
| Stale window IDs refresh once and never spawn | A closed window fails the keybind, or `cycle` launches an application | Unit, fake runner | `go test -run 'TestCycleFocusedRefreshes|TestCycleFocusedStops' ./...` |
| Shared retry extraction preserves activation behaviour | Refactoring changes the launch, retry, or error contract of `--app-id` | Unit, existing suite | `go test -run 'TestActivate' ./...` |
| Surface honours flags, arity, and exit codes | An unexpected argument is silently ignored, or help writes to stderr | Unit, CLI | `go test -run 'TestParseCycleOptions|TestExecuteCycle' ./...` |

## Readiness

Verdict: READY. The feature is additive, the selection rule is a pure function
over an existing data shape, no new external interface or privileged
environment is needed, and every acceptance criterion has a named test lane.
The one refactor touches code already pinned by exact-IPC-sequence tests.
