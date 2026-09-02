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
