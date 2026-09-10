# umbriel-raise

A small utility that focuses an existing application window
in [Umbriel](https://github.com/noctalia-dev/umbriel), or launches the
application when no matching window exists.

It uses Umbriel's own CLI for window discovery, cursor-warping focus, and
activation-aware launching. When several windows have the same app ID,
repeated invocations rotate through Umbriel's focus history.

## Features

- Exact, case-sensitive `app_id` matching
- Focuses and warps the cursor with Umbriel's runtime-switcher action
- MRU-aware rotation through multiple matching windows
- One keybind that cycles the windows of whichever app already has focus
- One retry when a window disappears between discovery and focus
- Launches through `umbriel msg spawn` to receive an activation token
- Preserves literal command arguments across Umbriel's shell boundary
- Interactive setup wizard for discovering app IDs and generating keybinds
- No runtime dependencies beyond Umbriel

## Installation

### Release binary

Download the binary for your architecture from the
[latest release](https://github.com/rigerc/umbriel-raise/releases/latest), then
install it somewhere on your `PATH`:

```bash
install -Dm755 umbriel-raise-linux-amd64 ~/.local/bin/umbriel-raise
```

Use `umbriel-raise-linux-arm64` instead on ARM64 systems.

### Go

With a recent Go toolchain installed:

```bash
go install github.com/rigerc/umbriel-raise@latest
```

## Usage

```text
umbriel-raise --app-id APP_ID -- COMMAND [ARG...]
umbriel-raise cycle [OPTIONS]
umbriel-raise setup [OPTIONS]
```

### Cycling the focused application

Bind `cycle` to rotate through the windows of whichever application currently
has focus:

```bash
umbriel-raise cycle
```

This is the "next terminal" keybind: focus any terminal, press the chord, and
focus moves to the next window of that same application. Press it while a
browser is focused and it cycles browser windows instead. No app ID and no
launch command are involved, so one chord covers every application.

Nothing happens when no window is focused or the focused application has only
one window, and `cycle` never launches anything.

### Guided setup

Run the terminal wizard while the application you want to configure is open:

```bash
umbriel-raise setup
```

The wizard groups running windows by exact `app_id`, lets you refresh or enter
an ID manually, and asks for the launch command and Umbriel key chord. It then
validates a temporary keybind config and prints both the standalone command and
a copyable TOML snippet. The active Umbriel config is never edited.

Generated actions use the absolute path of the running `umbriel-raise` binary,
which avoids compositor sessions with a narrower `PATH` than the login shell.
Use plain, line-oriented prompts when needed:

```bash
umbriel-raise setup --accessible
umbriel-raise setup --no-color
```

To save only the validated TOML keybind snippet, use `--output`. Existing files
are preserved unless `--force` is explicit:

```bash
umbriel-raise setup --output ~/.config/umbriel/raise-keybind.toml
umbriel-raise setup --output ./raise-keybind.toml --force
```

Use `--umbriel PATH` when the Umbriel CLI is not on the wizard's `PATH`.

### Manual setup

Find an application's ID with:

```bash
umbriel windows --json
```

Then configure an application:

```bash
umbriel-raise --app-id zen -- zen-browser
umbriel-raise --app-id com.mitchellh.ghostty -- ghostty
```

Use `--umbriel PATH` if the Umbriel CLI is not available as `umbriel` on your
`PATH`.

### Umbriel keybinds

Set `repeat = false` so holding a key does not trigger multiple activations:

```toml
[keybinds]
"Mod+B" = {
  action = "spawn:umbriel-raise --app-id zen -- zen-browser",
  repeat = false
}

"Mod+Return" = {
  action = "spawn:umbriel-raise --app-id com.mitchellh.ghostty -- ghostty",
  repeat = false
}

"Mod+Grave" = { action = "spawn:umbriel-raise cycle", repeat = false }
```

Validate the configuration after editing it:

```bash
umbriel validate
```

## Behavior

| Matching windows | Result |
| --- | --- |
| None | Launch the supplied command through Umbriel |
| One | Focus that window and warp the cursor to it |
| Several, another app active | Focus the most recently focused match |
| Several, a match active | Focus the least recently focused match, rotating through all matches |

The `cycle` subcommand reads the focused application from the window that
Umbriel reports as `active`, then applies the same table with launching removed:

| Windows of the focused app | Result |
| --- | --- |
| None focused, or no app ID | Do nothing |
| One | Do nothing; it is already focused |
| Several | Focus the least recently focused one, rotating through all of them |

Umbriel marks one window per workspace as `focused`, so only the `active` flag
identifies the window the user is looking at.

Matching uses the `app_id` field from `umbriel windows --json`; substrings and
regular expressions are not accepted. Umbriel returns windows in
most-recently-focused order, so rotation requires no state file. Launch command
arguments are treated literally, including whitespace and shell
metacharacters.

The command exits with status `0` on success, `1` when an Umbriel operation
fails, and `2` for invalid command-line usage. The setup wizard exits with
status `130` when canceled.

## Development

```bash
go test -race ./...
go vet ./...
go build ./...
```

Umbriel's [IPC documentation](https://docs.noctalia.dev/umbriel/ipc/) and
[action reference](https://docs.noctalia.dev/umbriel/actions/) describe the
commands used by this utility.
