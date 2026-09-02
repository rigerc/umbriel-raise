# PRD: Guided Rule Setup

Status: approved
Owner: umbriel-raise maintainers
Priority: P1
Last updated: 2026-09-02

## Requirements

- REQ-001: `umbriel-raise setup` starts the guided setup flow while all legacy
  invocations retain their current behavior.
- REQ-002: The flow discovers `umbriel windows --json`, groups windows by exact
  `app_id`, and displays sanitized app ID, representative title, window count,
  active state, and workspace when available.
- REQ-003: The user can select a discovered app, refresh discovery, or enter an
  app ID manually, then enter a launch command and key chord.
- REQ-004: The flow generates a standalone `umbriel-raise` command and an
  Umbriel table keybind with `repeat = false`, preserving shell and TOML syntax.
- REQ-005: The generated keybind is validated through a temporary config and
  the active config is never mutated.
- REQ-006: Generated actions use the resolved absolute executable path and
  failures include actionable Umbriel CLI diagnostics.
- REQ-007: The flow supports accessible/plain prompting and strips terminal
  control characters from compositor-provided display text.
- REQ-008: Output is printed by default. `--output` creates a file and refuses
  to overwrite an existing file unless `--force` is supplied.

## Acceptance Criteria

- AC-001: Existing grouped app IDs are available for selection from one window
  query, with duplicate app IDs collapsed and ordered deterministically.
- AC-002: An empty window list offers refresh and manual entry.
- AC-003: Window-query and JSON errors are actionable and perform no file write.
- AC-004: Generated actions preserve spaces, quotes, dollar signs, semicolons,
  backslashes, and Unicode in user-entered values.
- AC-005: The generated table keybind contains `repeat = false` and validates
  successfully before being printed or written.
- AC-006: Legacy `--app-id APP_ID -- COMMAND` routing remains unchanged.
- AC-007: Cancellation exits without writing a file.
- AC-008: Display labels cannot emit control or escape sequences supplied by a
  window title or app ID.
- AC-009: `--accessible`, `--no-color`, or a dumb terminal selects plain prompts.
- AC-010: Existing output files are preserved unless `--force` is explicit.

## UX States

- Loading: discovery happens before the selection prompt.
- Success: show the validated standalone command and keybind snippet.
- Empty: show refresh and manual-ID choices.
- Error: report the failing Umbriel operation and suggest `--umbriel PATH` when
  the executable cannot be found.
- Cancel: return a concise cancellation message and make no write.
- Accessibility: standard line-oriented prompts replace the full-screen TUI.
