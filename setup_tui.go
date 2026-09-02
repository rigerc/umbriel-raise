package main

import (
	"context"
	"errors"
	"io"
	"strings"

	"charm.land/huh/v2"
)

var errSetupCanceled = errors.New("setup canceled")

type huhSetupPrompter struct {
	input      io.Reader
	output     io.Writer
	accessible bool
}

type promptByteReader struct {
	reader io.Reader
}

func (r *promptByteReader) Read(buffer []byte) (int, error) {
	if len(buffer) > 1 {
		buffer = buffer[:1]
	}
	return r.reader.Read(buffer)
}

func newHuhSetupPrompter(input io.Reader, output io.Writer, accessible, noColor bool) *huhSetupPrompter {
	plain := accessible || noColor
	if plain {
		input = &promptByteReader{reader: input}
	}
	return &huhSetupPrompter{
		input:      input,
		output:     output,
		accessible: plain,
	}
}

func (p *huhSetupPrompter) ChooseApp(ctx context.Context, apps []appChoice) (string, error) {
	options := make([]huh.Option[string], 0, len(apps)+2)
	for _, app := range apps {
		options = append(options, huh.NewOption(app.Label(), app.AppID))
	}
	options = append(options,
		huh.NewOption("Enter an app ID manually", setupChoiceManual),
		huh.NewOption("Refresh running applications", setupChoiceRefresh),
	)

	var choice string
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Choose the application to raise").
			Description("App IDs come from the windows currently known to Umbriel.").
			Options(options...).
			Value(&choice),
	))
	if err := p.runForm(ctx, form); err != nil {
		return "", err
	}
	return choice, nil
}

func (p *huhSetupPrompter) ReadDetails(ctx context.Context, appID string) (setupDetails, error) {
	details := setupDetails{AppID: appID}
	fields := make([]huh.Field, 0, 3)
	if appID == "" {
		fields = append(fields, huh.NewInput().
			Title("Exact Umbriel app ID").
			Description("Run `umbriel windows --json` later if the app is not open yet.").
			Value(&details.AppID).
			Validate(requiredInput("app ID")))
	}
	fields = append(fields,
		huh.NewInput().
			Title("Launch command").
			Description("Shell command used only when no matching window exists.").
			Value(&details.LaunchCommand).
			Validate(requiredInput("launch command")),
		huh.NewInput().
			Title("Umbriel key chord").
			Description("For example: Mod+B or Mod+Shift+Return.").
			Value(&details.KeyChord).
			Validate(requiredInput("key chord")),
	)

	form := huh.NewForm(huh.NewGroup(fields...))
	if err := p.runForm(ctx, form); err != nil {
		return setupDetails{}, err
	}
	details.AppID = strings.TrimSpace(details.AppID)
	details.LaunchCommand = strings.TrimSpace(details.LaunchCommand)
	details.KeyChord = strings.TrimSpace(details.KeyChord)
	return details, nil
}

func requiredInput(name string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return errors.New(name + " is required")
		}
		return nil
	}
}

func (p *huhSetupPrompter) runForm(ctx context.Context, form *huh.Form) error {
	form.WithInput(p.input).
		WithOutput(p.output).
		WithAccessible(p.accessible)
	if p.accessible {
		form.WithTheme(huh.ThemeFunc(huh.ThemeBase))
	}

	if p.accessible {
		result := make(chan error, 1)
		go func() {
			result <- form.RunWithContext(ctx)
		}()
		select {
		case err := <-result:
			return normalizeSetupPromptError(ctx, err)
		case <-ctx.Done():
			return errSetupCanceled
		}
	}

	return normalizeSetupPromptError(ctx, form.RunWithContext(ctx))
}

func normalizeSetupPromptError(ctx context.Context, err error) error {
	if errors.Is(err, huh.ErrUserAborted) || ctx.Err() != nil {
		return errSetupCanceled
	}
	return err
}
