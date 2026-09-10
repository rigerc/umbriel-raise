# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Added

- Add `umbriel-raise cycle`, which rotates focus through the windows of
  whichever application currently owns focus, so a single keybind can switch
  between terminals, browser windows, or any other multi-window application.

## [0.2.0] - 2026-09-02

### Added

- Add `umbriel-raise setup`, an interactive terminal wizard that discovers
  running app IDs and generates a validated Umbriel keybind.
- Add accessible and no-color prompts plus safe optional snippet output with
  explicit overwrite protection.

## [0.1.1] - 2026-09-02

### Fixed

- Rotate reliably through matching windows using Umbriel's MRU and active-window
  state.
- Warp the cursor when raising a window, as expected for runtime switchers.
- Preserve literal launch arguments across Umbriel's shell-command boundary.

## [0.1.0] - 2026-09-01

### Added

- Initial Umbriel raise-or-launch utility.
