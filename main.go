package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

const usageText = `Usage: umbriel-raise --app-id APP_ID [--umbriel PATH] -- COMMAND [ARG...]

Focus an existing Umbriel window, or launch the command when none exists.

When several windows have the same app ID, repeated invocations rotate through
Umbriel's focus history. Focusing also warps the cursor to the selected window.
Matching is exact and case-sensitive.

Options:
`

type options struct {
	appID       string
	umbrielPath string
	command     []string
}

type window struct {
	ID      string `json:"id"`
	AppID   string `json:"app_id"`
	Title   string `json:"title"`
	Active  bool   `json:"active"`
	Focused bool   `json:"focused"`
}

type runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

type commandRunner struct {
	path string
}

func (r commandRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.path, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.Output()
	if err == nil {
		return stdout, nil
	}

	detail := strings.TrimSpace(stderr.String())
	if detail == "" {
		return nil, fmt.Errorf("run %s: %w", formatCommand(r.path, args), err)
	}

	return nil, fmt.Errorf("run %s: %w: %s", formatCommand(r.path, args), err, detail)
}

func formatCommand(path string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, path)
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%q", arg))
	}
	return strings.Join(parts, " ")
}

func formatShellCommand(command []string) string {
	parts := make([]string, len(command))
	for index, arg := range command {
		parts[index] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	return strings.Join(parts, " ")
}

func parseOptions(args []string, output io.Writer) (options, error) {
	var opts options
	flags := flag.NewFlagSet("umbriel-raise", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&opts.appID, "app-id", "", "exact app ID to match")
	flags.StringVar(&opts.umbrielPath, "umbriel", "umbriel", "path to the Umbriel CLI")
	flags.Usage = func() {
		_, _ = fmt.Fprint(output, usageText)
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return options{}, err
	}

	opts.command = flags.Args()
	if opts.appID == "" {
		return options{}, errors.New("--app-id is required")
	}
	if len(opts.command) == 0 {
		return options{}, errors.New("a launch command is required after --")
	}

	return opts, nil
}

func listWindows(ctx context.Context, r runner) ([]window, error) {
	output, err := r.Run(ctx, "windows", "--json")
	if err != nil {
		return nil, fmt.Errorf("list windows: %w", err)
	}

	var windows []window
	if err := json.Unmarshal(output, &windows); err != nil {
		return nil, fmt.Errorf("decode window list: %w", err)
	}

	return windows, nil
}

func matchingWindows(windows []window, appID string) []window {
	matches := make([]window, 0, len(windows))
	for _, candidate := range windows {
		if candidate.AppID == appID {
			matches = append(matches, candidate)
		}
	}
	return matches
}

func selectWindow(matches []window) (window, bool) {
	if len(matches) == 0 {
		return window{}, false
	}

	for _, candidate := range matches {
		if candidate.Active {
			return matches[len(matches)-1], true
		}
	}

	return matches[0], true
}

func focusWindow(ctx context.Context, r runner, id string) error {
	if _, err := r.Run(ctx, "msg", "window-focus-warp:"+id); err != nil {
		return fmt.Errorf("focus window %q: %w", id, err)
	}
	return nil
}

func spawn(ctx context.Context, r runner, command []string) error {
	if _, err := r.Run(ctx, "msg", "spawn", formatShellCommand(command)); err != nil {
		return fmt.Errorf("launch command: %w", err)
	}
	return nil
}

func activate(ctx context.Context, r runner, appID string, command []string) error {
	windows, err := listWindows(ctx, r)
	if err != nil {
		return err
	}

	selected, found := selectWindow(matchingWindows(windows, appID))
	if !found {
		return spawn(ctx, r, command)
	}

	initialFocusErr := focusWindow(ctx, r, selected.ID)
	if initialFocusErr == nil {
		return nil
	}

	// The selected window may have closed between the query and focus action.
	// Refresh once before returning an error or deciding that a launch is needed.
	windows, err = listWindows(ctx, r)
	if err != nil {
		return fmt.Errorf("refresh after focus failure (%v): %w", initialFocusErr, err)
	}

	selected, found = selectWindow(matchingWindows(windows, appID))
	if !found {
		return spawn(ctx, r, command)
	}

	if err := focusWindow(ctx, r, selected.ID); err != nil {
		return fmt.Errorf("focus failed after refresh (%v): %w", initialFocusErr, err)
	}

	return nil
}

func execute(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var flagOutput bytes.Buffer
	opts, err := parseOptions(args, &flagOutput)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = io.Copy(stdout, &flagOutput)
			return 0
		}
		if flagOutput.Len() > 0 {
			_, _ = io.Copy(stderr, &flagOutput)
		}
		_, _ = fmt.Fprintf(stderr, "umbriel-raise: %v\n\n", err)
		_, _ = fmt.Fprint(stderr, usageText)
		return 2
	}

	if err := activate(ctx, commandRunner{path: opts.umbrielPath}, opts.appID, opts.command); err != nil {
		_, _ = fmt.Fprintf(stderr, "umbriel-raise: %v\n", err)
		return 1
	}

	return 0
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(execute(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
