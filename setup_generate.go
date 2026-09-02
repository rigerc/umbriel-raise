package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type setupValues struct {
	AppID         string
	LaunchCommand string
	KeyChord      string
	Executable    string
}

type generatedSetup struct {
	Command string
	Snippet string
}

func generateSetup(values setupValues) (generatedSetup, error) {
	if strings.TrimSpace(values.AppID) == "" {
		return generatedSetup{}, errors.New("app ID is required")
	}
	launchCommand := strings.TrimSpace(values.LaunchCommand)
	if launchCommand == "" {
		return generatedSetup{}, errors.New("launch command is required")
	}
	keyChord := strings.TrimSpace(values.KeyChord)
	if keyChord == "" {
		return generatedSetup{}, errors.New("key chord is required")
	}
	if !filepath.IsAbs(values.Executable) {
		return generatedSetup{}, errors.New("umbriel-raise executable path must be absolute")
	}

	prefix := formatShellCommand([]string{values.Executable, "--app-id", values.AppID, "--"})
	command := prefix + " " + launchCommand
	action := "spawn:" + command
	snippet := fmt.Sprintf(
		"[keybinds]\n%s = { action = %s, repeat = false }\n",
		tomlBasicString(keyChord),
		tomlBasicString(action),
	)

	return generatedSetup{Command: command, Snippet: snippet}, nil
}

func tomlBasicString(value string) string {
	var output strings.Builder
	output.Grow(len(value) + 2)
	output.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\b':
			output.WriteString(`\b`)
		case '\t':
			output.WriteString(`\t`)
		case '\n':
			output.WriteString(`\n`)
		case '\f':
			output.WriteString(`\f`)
		case '\r':
			output.WriteString(`\r`)
		case '"':
			output.WriteString(`\"`)
		case '\\':
			output.WriteString(`\\`)
		default:
			if unicode.IsControl(r) {
				if r <= 0xffff {
					_, _ = fmt.Fprintf(&output, `\u%04X`, r)
				} else {
					_, _ = fmt.Fprintf(&output, `\U%08X`, r)
				}
				continue
			}
			output.WriteRune(r)
		}
	}
	output.WriteByte('"')
	return output.String()
}

func validateSetup(ctx context.Context, r runner, snippet, tempDir string) error {
	file, err := os.CreateTemp(tempDir, "umbriel-raise-*.toml")
	if err != nil {
		return fmt.Errorf("create temporary validation config: %w", err)
	}
	path := file.Name()
	defer func() {
		_ = os.Remove(path)
	}()

	if _, err := file.WriteString(snippet); err != nil {
		_ = file.Close()
		return fmt.Errorf("write temporary validation config: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary validation config: %w", err)
	}

	if _, err := r.Run(ctx, "validate", "-c", path); err != nil {
		return fmt.Errorf("validate generated keybind: %w", err)
	}
	return nil
}

func writeSetupOutput(path, content string, force bool) error {
	flags := os.O_WRONLY | os.O_CREATE
	if force {
		if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("refuse to overwrite non-regular output %q", path)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect output %q: %w", path, err)
		}
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}

	file, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("output %q already exists; use --force to overwrite it", path)
		}
		return fmt.Errorf("open output %q: %w", path, err)
	}

	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		return fmt.Errorf("write output %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close output %q: %w", path, err)
	}
	return nil
}
