package main

import (
	"context"
	"fmt"
	"io"
)

const (
	setupChoiceManual  = "\x00manual"
	setupChoiceRefresh = "\x00refresh"
)

type setupDetails struct {
	AppID         string
	LaunchCommand string
	KeyChord      string
}

type setupOptions struct {
	UmbrielPath string
	Accessible  bool
	NoColor     bool
	OutputPath  string
	Force       bool
}

type setupPrompter interface {
	ChooseApp(context.Context, []appChoice) (string, error)
	ReadDetails(context.Context, string) (setupDetails, error)
}

type setupRuntime struct {
	Runner     runner
	Prompter   setupPrompter
	Executable string
	TempDir    string
}

func runSetup(
	ctx context.Context,
	opts setupOptions,
	output io.Writer,
	runtime setupRuntime,
) error {
	var appID string
	for {
		apps, err := discoverApps(ctx, runtime.Runner)
		if err != nil {
			return err
		}

		choice, err := runtime.Prompter.ChooseApp(ctx, apps)
		if err != nil {
			return err
		}
		if choice == setupChoiceRefresh {
			continue
		}
		if choice != setupChoiceManual {
			appID = choice
		}
		break
	}

	details, err := runtime.Prompter.ReadDetails(ctx, appID)
	if err != nil {
		return err
	}
	generated, err := generateSetup(setupValues{
		AppID:         details.AppID,
		LaunchCommand: details.LaunchCommand,
		KeyChord:      details.KeyChord,
		Executable:    runtime.Executable,
	})
	if err != nil {
		return err
	}
	if err := validateSetup(ctx, runtime.Runner, generated.Snippet, runtime.TempDir); err != nil {
		return err
	}

	if opts.OutputPath != "" {
		if err := writeSetupOutput(opts.OutputPath, generated.Snippet, opts.Force); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(output, "\nSetup validated.\n\nStandalone command:\n%s\n\nUmbriel keybind:\n%s", generated.Command, generated.Snippet)
	if opts.OutputPath != "" {
		_, _ = fmt.Fprintf(output, "\nKeybind written to %s\n", opts.OutputPath)
	}
	return nil
}
