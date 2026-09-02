# BRD-lite: Guided Rule Setup

## Objective

Make first-time `umbriel-raise` configuration approachable without requiring
users to know how to discover an application's Umbriel `app_id` or how to
escape a keybind action by hand.

## Stakeholder and Outcome

The primary stakeholder is an Umbriel user configuring an application launcher.
Today they must inspect JSON, identify the correct app ID, and assemble TOML.
The desired state is a guided terminal flow that discovers running apps and
prints a validated, copyable configuration snippet.

Success means a user can generate a valid keybind from a running application in
under one minute, while the wizard never mutates the active Umbriel config.

## Scope

- Add an interactive `umbriel-raise setup` subcommand.
- Discover and group running windows by exact app ID.
- Collect the launch command and desired key chord.
- Preview a standalone command and an Umbriel keybind using the absolute
  `umbriel-raise` executable path.
- Validate the generated keybind through `umbriel validate`.
- Print by default, with an explicit safe file-output option.

## Non-goals

- Editing the user's active Umbriel configuration.
- Guessing launch commands from desktop entries.
- Launching applications, using the clipboard, or adding network/telemetry.
- Providing a graphical desktop interface.
