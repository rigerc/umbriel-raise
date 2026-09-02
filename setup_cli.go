package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/term"
)

const setupUsageText = `Usage: umbriel-raise setup [OPTIONS]

Interactively discover an Umbriel app ID and generate a validated raise-or-launch
keybind. The active Umbriel configuration is never modified.

Options:
`

func parseSetupOptions(args []string, output io.Writer) (setupOptions, error) {
	var opts setupOptions
	flags := flag.NewFlagSet("umbriel-raise setup", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&opts.UmbrielPath, "umbriel", "umbriel", "path to the Umbriel CLI")
	flags.BoolVar(&opts.Accessible, "accessible", false, "use line-oriented screen-reader-friendly prompts")
	flags.BoolVar(&opts.NoColor, "no-color", false, "disable color and use plain prompts")
	flags.StringVar(&opts.OutputPath, "output", "", "write the validated keybind snippet to this file")
	flags.BoolVar(&opts.Force, "force", false, "overwrite an existing regular output file")
	flags.Usage = func() {
		_, _ = fmt.Fprint(output, setupUsageText)
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return setupOptions{}, err
	}
	if flags.NArg() != 0 {
		return setupOptions{}, fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if opts.Force && opts.OutputPath == "" {
		return setupOptions{}, errors.New("--force requires --output")
	}
	return opts, nil
}

func runSetupCLI(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var flagOutput bytes.Buffer
	opts, err := parseSetupOptions(args, &flagOutput)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = io.Copy(stdout, &flagOutput)
			return 0
		}
		if flagOutput.Len() > 0 {
			_, _ = io.Copy(stderr, &flagOutput)
		}
		_, _ = fmt.Fprintf(stderr, "umbriel-raise setup: %v\n\n", err)
		_, _ = fmt.Fprint(stderr, setupUsageText)
		return 2
	}

	executable, err := resolvedExecutable()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "umbriel-raise setup: resolve executable: %v\n", err)
		return 1
	}

	accessible := opts.Accessible || opts.NoColor || strings.EqualFold(os.Getenv("TERM"), "dumb") || !isTerminalReader(stdin)
	err = runSetup(ctx, opts, stdout, setupRuntime{
		Runner:     commandRunner{path: opts.UmbrielPath},
		Prompter:   newHuhSetupPrompter(stdin, stderr, accessible, opts.NoColor),
		Executable: executable,
	})
	if errors.Is(err, errSetupCanceled) || errors.Is(err, context.Canceled) {
		_, _ = fmt.Fprintln(stderr, "umbriel-raise setup: canceled")
		return 130
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "umbriel-raise setup: %v\n", err)
		var executableError *exec.Error
		if errors.As(err, &executableError) {
			_, _ = fmt.Fprintf(stderr, "hint: install Umbriel or pass its CLI path with --umbriel PATH\n")
		}
		return 1
	}
	return 0
}

func resolvedExecutable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func isTerminalReader(reader io.Reader) bool {
	file, ok := reader.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(file.Fd())
}
