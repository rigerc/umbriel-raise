# Walkthrough: Correct Umbriel Raise Behavior

## Scope

Correct existing-window selection, focus behavior, and launch argument handling
against Umbriel 0.1.0 at revision `797415181382`.

## Acceptance Criteria

- AC-1: Existing matches use cursor-warping focus without launching.
- AC-2: MRU rotation reaches every matching window.
- AC-3: Workspace-local focus does not act as global active focus.
- AC-4: Launch arguments remain literal through Umbriel's shell boundary.
- AC-5: Existing matching, retry, launch, and CLI behavior remains intact.

## Evidence

| Check | Result | Evidence |
| --- | --- | --- |
| AC-1 focus action | PASS | `TestActivateFocusesOnlyMatchingWindow`; live focus of the active `zen` window |
| AC-2 MRU rotation | PASS | `TestSelectWindow/least_recent_when_a_match_is_active` with three candidates |
| AC-3 focus scope | PASS | `TestSelectWindow/workspace_local_focus_does_not_define_active_selection` |
| AC-4 literal arguments | PASS | `TestFormatShellCommandPreservesArguments`; live launch created exactly one whitespace-bearing filename |
| AC-5 regression suite | PASS | `go test -count=1 -race ./...`, `go vet ./...`, and `go build ./...` |

## Risks

The live session had one window for the selected application, so three-window
rotation was verified at the pure selection layer rather than by creating
disruptive extra GUI windows. The selection fixture models Umbriel's documented
MRU snapshots directly.

## Outcome Report

feature_status: implemented
requirement_trace: BRD-OBJ-1 -> REQ-1 -> AC-1..AC-5 -> selection/focus/spawn contracts -> unit and live evidence
completed_evidence: [focused RED-GREEN tests, race suite, vet, build, live focus, live quoted spawn]; missing_evidence: []; decision_needed: []; recommended_next_workflow: deploy-release

## Next Workflow

deploy-release

---

# Walkthrough: Guided Rule Setup

## Scope

Add `umbriel-raise setup` for interactive app-ID discovery and validated
keybind generation without editing the active Umbriel config.

## Acceptance Evidence

| Requirement | Result | Evidence |
| --- | --- | --- |
| REQ-001 legacy-safe routing | PASS | `TestExecuteRoutesSetup` plus the full existing activation suite |
| REQ-002 discovery/grouping | PASS | `TestDiscoverAppsGroupsAndSortsWindows`; live Ghostty/T3 Code/Zen list |
| REQ-003 guided inputs | PASS | discovered, refresh/manual, and accessible Huh flow tests |
| REQ-004 safe generation | PASS | special-character generation test and live keybind preview |
| REQ-005 validation | PASS | temp-config boundary tests and installed `umbriel validate -c` |
| REQ-006 absolute path | PASS | generated live action used the resolved test binary path |
| REQ-007 accessible/safe display | PASS | no-color end-to-end test and hostile control-character fixture |
| REQ-008 safe output | PASS | exclusive-create and explicit-force overwrite tests |

## Quality Gates

- `go test -count=1 -race ./...`: PASS.
- `go vet ./...`: PASS.
- `go build ./...`: PASS.
- `gopls check` for all Go files: PASS.
- `git diff --check`: PASS.
- Statement coverage: 79.0%; uncovered code is primarily full-screen terminal
  rendering and OS-level error branches, while the pure and external-boundary
  setup contracts have direct coverage.

## Outcome Report

feature_status: implemented
requirement_trace: BRD guided-setup objective -> REQ-001..REQ-008 -> AC-001..AC-010 -> routing/discovery/generation/interaction tests and live validation
completed_evidence: [focused RED-GREEN slices, accessible CLI E2E, race suite, vet, build, gopls, live Umbriel validation]; missing_evidence: []; decision_needed: []; recommended_next_workflow: verify-work

## Next Workflow

verify-work

---

# Walkthrough: Cycle Windows of the Focused Application

## Scope

Add `umbriel-raise cycle`, rotating focus through the windows of whichever
application currently owns focus, against Umbriel 0.1.0.

## Acceptance Criteria

- AC-1: `cycle` routes without disturbing legacy activation or `setup`.
- AC-2: The focused application comes from global `active`, not workspace-local
  `focused`.
- AC-3: No active window, no app ID, or a single window is a silent success.
- AC-4: Several windows rotate to the least recently focused match.
- AC-5: A stale window ID refreshes once and never launches an application.
- AC-6: Existing activation, setup, and CLI behavior remains intact.

## Evidence

| Check | Result | Evidence |
| --- | --- | --- |
| AC-1 routing | PASS | `TestExecuteRoutesCycle`; existing `TestExecuteRoutesSetup` and legacy usage tests |
| AC-2 focus scope | PASS | `TestSelectCycleTarget/workspace_local_focus_does_not_select_the_app` |
| AC-3 quiet no-ops | PASS | `TestSelectCycleTarget` no-active/no-app-ID/single-window cases; `TestCycleFocusedIsQuietWhenFocusedAppHasOneWindow` asserts no focus call is issued; live no-op on a single-window `emacs` |
| AC-4 rotation | PASS | `TestSelectCycleTarget` two- and three-window cases; `TestCycleFocusedRotatesToLeastRecentMatch`; live rotation across five `kitty` windows reached three distinct windows without repeating |
| AC-5 stale retry | PASS | `TestCycleFocusedRefreshesAfterStaleWindowID`, `TestCycleFocusedStopsWhenTargetDisappears`, `TestCycleFocusedReportsFocusFailure` |
| AC-6 regression suite | PASS | `go test -count=1 -race ./...`, `go vet ./...`, `go build ./...`, `gofmt -l`; live `--app-id emacs` focused an existing window with the window count unchanged |

Statement coverage: every function in `cycle.go` and the extracted
`focusSelected` helper reach 100%, moving the package from 80.4% to 84.2%. The
command layer is covered without a compositor by pointing `--umbriel` at a
stub script for the success path and at a missing path for the failure path.

## Risks

`activate` and `cycle` now share one `focusSelected` retry helper. The existing
activation tests assert exact IPC call sequences for the launch, focus, retry,
and refresh paths, so they pin the extraction; they pass unchanged.

`selectCycleTarget` defers to `selectWindow`, which returns the last match when
any match is active. This assumes Umbriel lists the globally active window
first among its own application's windows, which is the same MRU assumption the
existing rotation already relies on. If that ordering were violated, `cycle`
would re-focus the current window and warp the cursor to it rather than
misbehave. Live rotation across five windows showed the assumption holding.

## Outcome Report

feature_status: implemented
requirement_trace: focused-app rotation objective -> REQ-001..REQ-007 -> AC-1..AC-6 -> routing/selection/activation/surface contracts -> unit and live evidence
completed_evidence: [RED-GREEN focused tests, race suite, vet, build, gofmt, live rotation, live no-op, live legacy regression]; missing_evidence: []; decision_needed: []; recommended_next_workflow: deploy-release

## Next Workflow

deploy-release
