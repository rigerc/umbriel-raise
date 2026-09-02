# Implementation Plan: Guided Rule Setup

## Architecture

Keep the existing single-binary CLI and add a narrowly routed `setup`
subcommand. Separate compositor discovery, pure snippet generation, filesystem
output, and Huh-based interaction so the externally risky boundaries can be
tested without driving a terminal UI.

## Task Slices

| Slice | Requirements | Verification |
| --- | --- | --- |
| Routing | REQ-001 | Setup dispatch test plus existing CLI suite |
| Discovery | REQ-002, REQ-006, REQ-007 | Grouping, ordering, sanitization, and failure tests |
| Generation | REQ-004, REQ-005, REQ-008 | Shell/TOML round trips, validator fake, safe-write tests |
| Interaction | REQ-003, REQ-007 | Accessible scripted flows for select/manual/refresh/cancel |

## Interfaces and Failure Modes

- No external API, database, migration, permissions, or background process.
- Read interface: `umbriel windows --json`.
- Validation interface: `umbriel validate -c TEMP_CONFIG`.
- Optional write interface: a user-selected path created with exclusive-create
  semantics unless force is explicit.
- Failures: missing CLI, compositor IPC error, malformed JSON, invalid inputs,
  validation rejection, cancellation, and filesystem errors.

## Non-functional Constraints

- One discovery command per refresh; expected prompt readiness under one second
  when Umbriel responds normally.
- No active-config mutation and no automatic launch command execution.
- Control characters from IPC data never reach interactive display labels.
- All existing activation tests continue to pass under the race detector.

## Requirement Trace

BRD guided-setup objective -> REQ-001..REQ-008 -> AC-001..AC-010 -> routing,
discovery, generation/output, and accessible-flow tests.
