package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
)

const cycleUsageText = `Usage: umbriel-raise cycle [--umbriel PATH]

Rotate focus to the next window of the application that currently owns focus.

Repeated invocations walk every window of that application using Umbriel's
focus history, warping the cursor to each one. Nothing happens when no window
is focused or the focused application has a single window, and no application
is ever launched.

Options:
`

type cycleOptions struct {
	umbrielPath string
}

// activeWindow returns the window holding global focus. Umbriel marks one
// window per workspace as focused, so only the active flag identifies the
// window the user is actually looking at.
func activeWindow(windows []window) (window, bool) {
	for _, candidate := range windows {
		if candidate.Active {
			return candidate, true
		}
	}
	return window{}, false
}

// selectCycleTarget picks the next window of the application that owns focus.
// It reports false when nothing is active, when the active window carries no
// app ID, or when that application has no second window to rotate to.
func selectCycleTarget(windows []window) (window, bool) {
	active, found := activeWindow(windows)
	if !found || active.AppID == "" {
		return window{}, false
	}

	matches := matchingWindows(windows, active.AppID)
	if len(matches) < 2 {
		return window{}, false
	}

	return selectWindow(matches)
}

func cycleFocused(ctx context.Context, r runner) error {
	return focusSelected(ctx, r, selectCycleTarget, func() error { return nil })
}

func parseCycleOptions(args []string, output io.Writer) (cycleOptions, error) {
	var opts cycleOptions
	flags := flag.NewFlagSet("umbriel-raise cycle", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&opts.umbrielPath, "umbriel", "umbriel", "path to the Umbriel CLI")
	flags.Usage = func() {
		_, _ = fmt.Fprint(output, cycleUsageText)
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return cycleOptions{}, err
	}
	if flags.NArg() != 0 {
		return cycleOptions{}, fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}

	return opts, nil
}

func runCycleCLI(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var flagOutput bytes.Buffer
	opts, err := parseCycleOptions(args, &flagOutput)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = io.Copy(stdout, &flagOutput)
			return 0
		}
		if flagOutput.Len() > 0 {
			_, _ = io.Copy(stderr, &flagOutput)
		}
		_, _ = fmt.Fprintf(stderr, "umbriel-raise cycle: %v\n\n", err)
		_, _ = fmt.Fprint(stderr, cycleUsageText)
		return 2
	}

	if err := cycleFocused(ctx, commandRunner{path: opts.umbrielPath}); err != nil {
		_, _ = fmt.Fprintf(stderr, "umbriel-raise cycle: %v\n", err)
		return 1
	}

	return 0
}
