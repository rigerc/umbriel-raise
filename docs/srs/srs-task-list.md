# Task: Correct Umbriel Raise Behavior

## Scope

Implement AC-1 through AC-5 from the approved implementation plan without
adding runtime dependencies.

## Checklist

- [x] Add active/MRU selection regression tests and implementation.
- [x] Add focus-warp regression coverage and implementation.
- [x] Add shell-quoting regression tests and implementation.
- [x] Update user documentation.
- [x] Run focused and full verification gates.
- [x] Record the verified walkthrough and release plan.

## Decisions

- The default runtime-switcher action is `window-focus-warp`.
- Cycling uses Umbriel's MRU ordering and requires no persistent state.
- Launch arguments retain argv semantics at the shell boundary.

## Evidence

### Test Intent Records

| Contract | Fault | Layer | Cases | Command |
| --- | --- | --- | --- | --- |
| Selection raises the MRU match when another app is active and rotates to the LRU match when the app is active | Workspace-local focus or a mutable MRU list skips/toggles matching windows | Unit; selection is pure | no matches, one match, inactive matches, active match with three candidates, multiple workspace-focused matches | `go test -run 'TestSelectWindow$' ./...` |
| Raising an existing match uses Umbriel's cursor-warp focus action | Focus-only IPC lets pointer focus reclaim the old window and does not behave as a runtime switcher | Unit at the runner boundary; the emitted action is the application-owned contract | existing match | `go test -run 'TestActivateFocusesOnlyMatchingWindow$' ./...` |
| Launch commands preserve every argv element through Umbriel's `/bin/sh -c` boundary | Joining raw arguments changes whitespace/empty values or executes metacharacters | Integration at the real POSIX shell boundary | simple, spaces, empty, quote, dollar/semicolon, newline | `go test -run 'TestFormatShellCommandPreservesArguments$' ./...` |

### Verification

- Focused selection, focus-action, and shell-boundary tests completed their
  RED-GREEN loops.
- `go test -count=1 -race ./...`: PASS.
- `go vet ./...`: PASS.
- `go build ./...`: PASS.
- Live existing-window focus against Umbriel 0.1.0: PASS.
- Live no-match launch with a whitespace-bearing argument: PASS.

## Next Workflow

verify-work
