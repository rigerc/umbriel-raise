package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeSetupPrompter struct {
	t       *testing.T
	choices []string
	details setupDetails
	calls   int
}

func TestHuhSetupPrompterAccessibleFlow(t *testing.T) {
	input := strings.NewReader("1\nzen-browser --private-window\nMod+B\n")
	var output strings.Builder
	prompter := newHuhSetupPrompter(input, &output, true, false)

	appID, err := prompter.ChooseApp(context.Background(), []appChoice{{
		AppID: "zen", Title: "Browser", Workspace: "DP-1:1", Count: 1,
	}})
	if err != nil {
		t.Fatalf("ChooseApp() error = %v", err)
	}
	details, err := prompter.ReadDetails(context.Background(), appID)
	if err != nil {
		t.Fatalf("ReadDetails() error = %v", err)
	}

	want := setupDetails{AppID: "zen", LaunchCommand: "zen-browser --private-window", KeyChord: "Mod+B"}
	if !reflect.DeepEqual(details, want) {
		t.Fatalf("details = %#v, want %#v", details, want)
	}
	if strings.Contains(output.String(), "\x1b") {
		t.Fatalf("plain output contains an escape sequence: %q", output.String())
	}
}

func TestSetupCLIHelp(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := executeWithSetup(
		context.Background(),
		[]string{"setup", "--help"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		runSetupCLI,
	)
	if code != 0 {
		t.Fatalf("executeWithSetup() = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage: umbriel-raise setup") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestSetupCLIRejectsForceWithoutOutput(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := executeWithSetup(
		context.Background(),
		[]string{"setup", "--force"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		runSetupCLI,
	)
	if code != 2 || !strings.Contains(stderr.String(), "--force requires --output") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestSetupCLIEndToEndAccessible(t *testing.T) {
	umbrielPath := filepath.Join(t.TempDir(), "umbriel")
	script := `#!/bin/sh
case "$1" in
  windows)
    printf '%s\n' '[{"id":"1","app_id":"zen","title":"Browser","workspace":"DP-1:1"}]'
    ;;
  validate)
    test "$2" = "-c" && grep -q 'repeat = false' "$3"
    ;;
  *)
    exit 64
    ;;
esac
`
	if err := os.WriteFile(umbrielPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake Umbriel: %v", err)
	}

	input := strings.NewReader("1\nzen-browser --private-window\nMod+B\n")
	var stdout strings.Builder
	var stderr strings.Builder
	code := executeWithSetup(
		context.Background(),
		[]string{"setup", "--accessible", "--no-color", "--umbriel", umbrielPath},
		input,
		&stdout,
		&stderr,
		runSetupCLI,
	)
	if code != 0 {
		t.Fatalf("executeWithSetup() = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Setup validated") || !strings.Contains(stdout.String(), "repeat = false") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "\x1b") {
		t.Fatalf("plain prompts contain escape sequences: %q", stderr.String())
	}
}

type canceledSetupPrompter struct{}

func (canceledSetupPrompter) ChooseApp(context.Context, []appChoice) (string, error) {
	return "", errSetupCanceled
}

func (canceledSetupPrompter) ReadDetails(context.Context, string) (setupDetails, error) {
	return setupDetails{}, errSetupCanceled
}

func TestRunSetupWizardCancellationDoesNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "should-not-exist.toml")
	runner := &setupFlowRunner{t: t, windowsOutput: `[]`}

	err := runSetup(context.Background(), setupOptions{OutputPath: path}, io.Discard, setupRuntime{
		Runner:     runner,
		Prompter:   canceledSetupPrompter{},
		Executable: "/bin/umbriel-raise",
		TempDir:    t.TempDir(),
	})
	if !errors.Is(err, errSetupCanceled) {
		t.Fatalf("runSetup() error = %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output exists after cancellation: %v", statErr)
	}
}

func (p *fakeSetupPrompter) ChooseApp(_ context.Context, _ []appChoice) (string, error) {
	p.t.Helper()
	if p.calls >= len(p.choices) {
		p.t.Fatal("unexpected app choice prompt")
	}
	choice := p.choices[p.calls]
	p.calls++
	return choice, nil
}

func (p *fakeSetupPrompter) ReadDetails(_ context.Context, appID string) (setupDetails, error) {
	p.t.Helper()
	details := p.details
	if appID != "" {
		details.AppID = appID
	}
	return details, nil
}

type setupFlowRunner struct {
	t             *testing.T
	windowsOutput string
	validated     string
	windowCalls   int
}

func (r *setupFlowRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.t.Helper()
	switch {
	case reflect.DeepEqual(args, []string{"windows", "--json"}):
		r.windowCalls++
		return []byte(r.windowsOutput), nil
	case len(args) == 3 && args[0] == "validate" && args[1] == "-c":
		data, err := os.ReadFile(args[2])
		if err != nil {
			r.t.Fatalf("read validation config: %v", err)
		}
		r.validated = string(data)
		return nil, nil
	default:
		r.t.Fatalf("unexpected runner call: %v", args)
		return nil, nil
	}
}

func TestRunSetupWizardGeneratesValidatedPreview(t *testing.T) {
	runner := &setupFlowRunner{
		t:             t,
		windowsOutput: `[{"id":"1","app_id":"zen","title":"Browser","workspace":"DP-1:1"}]`,
	}
	prompter := &fakeSetupPrompter{
		t:       t,
		choices: []string{"zen"},
		details: setupDetails{LaunchCommand: "zen-browser --private-window", KeyChord: "Mod+B"},
	}
	var output strings.Builder

	err := runSetup(context.Background(), setupOptions{}, &output, setupRuntime{
		Runner:     runner,
		Prompter:   prompter,
		Executable: "/home/user/.local/bin/umbriel-raise",
		TempDir:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("runSetup() error = %v", err)
	}
	if runner.windowCalls != 1 || runner.validated == "" {
		t.Fatalf("window calls = %d, validated = %q", runner.windowCalls, runner.validated)
	}
	for _, want := range []string{"Standalone command", "'/home/user/.local/bin/umbriel-raise'", "[keybinds]", "repeat = false"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output = %q, missing %q", output.String(), want)
		}
	}
}

func TestRunSetupWizardRefreshesThenAcceptsManualAppID(t *testing.T) {
	runner := &setupFlowRunner{t: t, windowsOutput: `[]`}
	prompter := &fakeSetupPrompter{
		t:       t,
		choices: []string{setupChoiceRefresh, setupChoiceManual},
		details: setupDetails{AppID: "org.example.App", LaunchCommand: "example-app", KeyChord: "Mod+E"},
	}

	err := runSetup(context.Background(), setupOptions{}, io.Discard, setupRuntime{
		Runner:     runner,
		Prompter:   prompter,
		Executable: "/bin/umbriel-raise",
		TempDir:    t.TempDir(),
	})
	if err != nil {
		t.Fatalf("runSetup() error = %v", err)
	}
	if runner.windowCalls != 2 {
		t.Fatalf("window calls = %d, want 2", runner.windowCalls)
	}
	if !strings.Contains(runner.validated, "org.example.App") {
		t.Fatalf("validated config = %q", runner.validated)
	}
}

func TestGenerateSetupPreservesSpecialCharacters(t *testing.T) {
	generated, err := generateSetup(setupValues{
		AppID:         "weird 'app'",
		LaunchCommand: `printf '%s\n' "a b" '$HOME; literal'`,
		KeyChord:      `Mod+Shift+"B"`,
		Executable:    "/opt/Umbriel Raise/bin",
	})
	if err != nil {
		t.Fatalf("generateSetup() error = %v", err)
	}

	wantCommand := `'/opt/Umbriel Raise/bin' '--app-id' 'weird '\''app'\''' '--' printf '%s\n' "a b" '$HOME; literal'`
	if generated.Command != wantCommand {
		t.Fatalf("Command = %q, want %q", generated.Command, wantCommand)
	}
	if !strings.Contains(generated.Snippet, `"Mod+Shift+\"B\"" = { action = "spawn:`) {
		t.Fatalf("Snippet does not TOML-escape key/action: %q", generated.Snippet)
	}
	if !strings.Contains(generated.Snippet, `repeat = false`) {
		t.Fatalf("Snippet does not disable repeat: %q", generated.Snippet)
	}
}

func TestGenerateSetupRejectsIncompleteValues(t *testing.T) {
	tests := []struct {
		name   string
		values setupValues
	}{
		{name: "app id", values: setupValues{LaunchCommand: "zen", KeyChord: "Mod+B", Executable: "/bin/raise"}},
		{name: "launch command", values: setupValues{AppID: "zen", KeyChord: "Mod+B", Executable: "/bin/raise"}},
		{name: "key chord", values: setupValues{AppID: "zen", LaunchCommand: "zen", Executable: "/bin/raise"}},
		{name: "absolute executable", values: setupValues{AppID: "zen", LaunchCommand: "zen", KeyChord: "Mod+B", Executable: "raise"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := generateSetup(test.values); err == nil {
				t.Fatal("generateSetup() error = nil")
			}
		})
	}
}

type validatingRunner struct {
	t       *testing.T
	want    string
	content string
	err     error
}

func (r *validatingRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.t.Helper()
	if len(args) != 3 || args[0] != "validate" || args[1] != "-c" {
		r.t.Fatalf("validate args = %v", args)
	}
	data, err := os.ReadFile(args[2])
	if err != nil {
		r.t.Fatalf("read temporary config: %v", err)
	}
	r.content = string(data)
	if r.content != r.want {
		r.t.Fatalf("temporary config = %q, want %q", r.content, r.want)
	}
	return nil, r.err
}

func TestValidateSetupUsesTemporaryConfig(t *testing.T) {
	runner := &validatingRunner{t: t, want: "[keybinds]\n\"Mod+B\" = { action = \"spawn:true\", repeat = false }\n"}
	if err := validateSetup(context.Background(), runner, runner.want, t.TempDir()); err != nil {
		t.Fatalf("validateSetup() error = %v", err)
	}
}

func TestValidateSetupReportsUmbrielRejection(t *testing.T) {
	runner := &validatingRunner{t: t, want: "invalid", err: errors.New("bad keybind")}
	err := validateSetup(context.Background(), runner, runner.want, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "bad keybind") {
		t.Fatalf("validateSetup() error = %v", err)
	}
}

func TestWriteSetupOutputRefusesOverwriteWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "umbriel-raise.toml")
	if err := os.WriteFile(path, []byte("keep me"), 0o600); err != nil {
		t.Fatalf("seed output: %v", err)
	}

	err := writeSetupOutput(path, "replace me", false)
	if err == nil {
		t.Fatal("writeSetupOutput() error = nil")
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read output: %v", readErr)
	}
	if string(data) != "keep me" {
		t.Fatalf("output = %q, want preserved content", data)
	}
}

func TestWriteSetupOutputCanForceOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "umbriel-raise.toml")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("seed output: %v", err)
	}

	if err := writeSetupOutput(path, "new", true); err != nil {
		t.Fatalf("writeSetupOutput() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("output = %q, want new", data)
	}
}

func TestDiscoverAppsGroupsAndSortsWindows(t *testing.T) {
	runner := &fakeRunner{
		t:    t,
		want: [][]string{{"windows", "--json"}},
		replies: []response{{output: `[
			{"id":"3","app_id":"zen","title":"Second","workspace":"DP-1:2"},
			{"id":"1","app_id":"code","title":"Editor","workspace":"DP-1:1","active":true},
			{"id":"2","app_id":"zen","title":"Browser","workspace":"DP-1:1","active":true},
			{"id":"4","app_id":"","title":"Desktop"}
		]`}},
	}

	got, err := discoverApps(context.Background(), runner)
	if err != nil {
		t.Fatalf("discoverApps() error = %v", err)
	}
	runner.verify(t)

	want := []appChoice{
		{AppID: "code", Title: "Editor", Workspace: "DP-1:1", Count: 1, Active: true},
		{AppID: "zen", Title: "Second", Workspace: "DP-1:2", Count: 2, Active: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverApps() = %#v, want %#v", got, want)
	}
}

func TestDiscoverAppsSanitizesTerminalText(t *testing.T) {
	runner := &fakeRunner{
		t:       t,
		want:    [][]string{{"windows", "--json"}},
		replies: []response{{output: `[{"id":"1","app_id":"ze\u001b[31mn","title":"bad\n\ttitle\u202ereversed","workspace":"DP-1\u009b2"}]`}},
	}

	apps, err := discoverApps(context.Background(), runner)
	if err != nil {
		t.Fatalf("discoverApps() error = %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("discoverApps() returned %d choices", len(apps))
	}

	label := apps[0].Label()
	for _, unsafe := range []string{"\x1b", "\n", "\t", "\u009b", "\u202e"} {
		if strings.Contains(label, unsafe) {
			t.Fatalf("Label() = %q, contains control %q", label, unsafe)
		}
	}
}
