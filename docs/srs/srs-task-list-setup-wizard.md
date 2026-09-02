# Task: Guided Rule Setup

## Checklist

- [x] Route `setup` without changing legacy activation.
- [x] Discover, sanitize, group, and sort running applications.
- [x] Generate shell-safe and TOML-safe previews using the absolute binary path.
- [x] Validate generated keybinds in a temporary config.
- [x] Add accessible Huh prompts for discovery, refresh/manual entry, and details.
- [x] Implement print-by-default and safe optional file output.
- [x] Update README/help and record verification evidence.

## Test Intent Records

| Contract | Distinct fault | Layer | Focused command |
| --- | --- | --- | --- |
| `setup` dispatches while legacy args do not | Subcommand is parsed as a missing `--app-id` or legacy calls are intercepted | Unit, CLI routing | `go test -run 'TestExecuteRoutesSetup$' ./...` |
| Discovery yields safe deterministic app choices | Duplicate app IDs, unstable order, or IPC escape bytes confuse selection | Unit, pure grouping | `go test -run 'TestDiscoverApps' ./...` |
| Preview is valid shell/TOML with `repeat = false` | Special characters change argv or break the keybind | Unit plus shell boundary | `go test -run 'TestGenerateSetup' ./...` |
| Validation and output are non-mutating by default | Invalid snippets are emitted or an existing file is overwritten | Boundary unit tests | `go test -run 'TestValidateSetup|TestWriteSetup' ./...` |
| Plain wizard handles discovered/manual/refresh/cancel states | Empty discovery traps the user or cancellation writes output | Component, scripted accessible I/O | `go test -run 'TestRunSetupWizard' ./...` |

## Readiness

Verdict: READY. The feature is additive, requirements and failure modes are
stable, UX covers loading/empty/error/cancel/accessibility states, no migration
or privileged environment is needed, and all acceptance criteria have named
test lanes.

## Implementation Evidence

- Routing RED: `executeWithSetup` was undefined; focused routing and legacy
  usage tests passed after the additive dispatcher was implemented.
- Discovery RED: `discoverApps` and `appChoice` were undefined; grouping,
  ordering, empty-ID handling, and terminal sanitization tests now pass.
- Generation RED: setup generation, validation, and safe-write contracts were
  undefined; special-character, validator, and overwrite tests now pass.
- Interaction RED: wizard orchestration and Huh adapter were undefined; the
  discovered, refresh/manual, accessible/no-color, cancellation, and CLI
  end-to-end flows now pass.
- Fresh gates: `go test -count=1 -race ./...`, `go vet ./...`, `go build ./...`,
  `gopls check` on all Go files, and `git diff --check` passed.
- Live gate: the no-color wizard discovered three current applications and the
  installed Umbriel 0.1.0 validator accepted the generated Zen keybind.

## Outcome

Status: implemented. Next workflow: `verify-work`.
