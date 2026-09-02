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
