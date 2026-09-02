package main

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestExecuteRoutesSetup(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	called := false

	setup := func(_ context.Context, args []string, stdin io.Reader, gotStdout, gotStderr io.Writer) int {
		called = true
		if !reflect.DeepEqual(args, []string{"--accessible", "--umbriel", "/opt/umbriel"}) {
			t.Fatalf("setup args = %v", args)
		}
		if stdin == nil || gotStdout != &stdout || gotStderr != &stderr {
			t.Fatal("setup streams were not preserved")
		}
		return 7
	}

	code := executeWithSetup(
		context.Background(),
		[]string{"setup", "--accessible", "--umbriel", "/opt/umbriel"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		setup,
	)
	if code != 7 || !called {
		t.Fatalf("executeWithSetup() = %d, called = %v", code, called)
	}
}

type response struct {
	output string
	err    error
}

type fakeRunner struct {
	t       *testing.T
	want    [][]string
	replies []response
	calls   int
}

func (r *fakeRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.t.Helper()
	if r.calls >= len(r.want) {
		r.t.Fatalf("unexpected call %d: %v", r.calls+1, args)
	}
	if !reflect.DeepEqual(args, r.want[r.calls]) {
		r.t.Fatalf("call %d = %v, want %v", r.calls+1, args, r.want[r.calls])
	}

	reply := r.replies[r.calls]
	r.calls++
	return []byte(reply.output), reply.err
}

func (r *fakeRunner) verify(t *testing.T) {
	t.Helper()
	if r.calls != len(r.want) {
		t.Fatalf("received %d calls, want %d", r.calls, len(r.want))
	}
}

func TestMatchingWindowsUsesExactAppID(t *testing.T) {
	windows := []window{
		{ID: "one", AppID: "zen"},
		{ID: "two", AppID: "Zen"},
		{ID: "three", AppID: "zen-browser"},
	}

	got := matchingWindows(windows, "zen")
	want := []window{{ID: "one", AppID: "zen"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("matchingWindows() = %#v, want %#v", got, want)
	}
}

func TestSelectWindow(t *testing.T) {
	tests := []struct {
		name    string
		matches []window
		wantID  string
		found   bool
	}{
		{name: "empty", found: false},
		{name: "one", matches: []window{{ID: "one"}}, wantID: "one", found: true},
		{
			name:    "most recent when another app is active",
			matches: []window{{ID: "one"}, {ID: "two"}},
			wantID:  "one",
			found:   true,
		},
		{
			name:    "least recent when a match is active",
			matches: []window{{ID: "one", Active: true}, {ID: "two"}, {ID: "three"}},
			wantID:  "three",
			found:   true,
		},
		{
			name: "workspace local focus does not define active selection",
			matches: []window{
				{ID: "one", Focused: true},
				{ID: "two", Focused: true},
			},
			wantID: "one",
			found:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, found := selectWindow(test.matches)
			if found != test.found || got.ID != test.wantID {
				t.Fatalf("selectWindow() = (%q, %v), want (%q, %v)", got.ID, found, test.wantID, test.found)
			}
		})
	}
}

func TestActivateSpawnsWhenNoWindowMatches(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "spawn", `'zen-browser' '--private-window'`},
		},
		replies: []response{
			{output: `[{"id":"terminal","app_id":"com.mitchellh.ghostty","focused":true}]`},
			{},
		},
	}

	err := activate(context.Background(), runner, "zen", []string{"zen-browser", "--private-window"})
	if err != nil {
		t.Fatalf("activate() error = %v", err)
	}
	runner.verify(t)
}

func TestActivateFocusesOnlyMatchingWindow(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "window-focus-warp:browser"},
		},
		replies: []response{
			{output: `[{"id":"browser","app_id":"zen","focused":false}]`},
			{},
		},
	}

	err := activate(context.Background(), runner, "zen", []string{"zen-browser"})
	if err != nil {
		t.Fatalf("activate() error = %v", err)
	}
	runner.verify(t)
}

func TestActivateCyclesMatchingWindows(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "window-focus-warp:second"},
		},
		replies: []response{
			{output: `[
				{"id":"first","app_id":"zen","active":true,"focused":true},
				{"id":"terminal","app_id":"com.mitchellh.ghostty","focused":false},
				{"id":"second","app_id":"zen","focused":false}
			]`},
			{},
		},
	}

	err := activate(context.Background(), runner, "zen", []string{"zen-browser"})
	if err != nil {
		t.Fatalf("activate() error = %v", err)
	}
	runner.verify(t)
}

func TestActivateRefreshesAfterStaleWindowID(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "window-focus-warp:stale"},
			{"windows", "--json"},
			{"msg", "window-focus-warp:replacement"},
		},
		replies: []response{
			{output: `[{"id":"stale","app_id":"zen"}]`},
			{err: errors.New("unknown window")},
			{output: `[{"id":"replacement","app_id":"zen"}]`},
			{},
		},
	}

	err := activate(context.Background(), runner, "zen", []string{"zen-browser"})
	if err != nil {
		t.Fatalf("activate() error = %v", err)
	}
	runner.verify(t)
}

func TestActivateSpawnsWhenWindowDisappears(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "window-focus-warp:stale"},
			{"windows", "--json"},
			{"msg", "spawn", `'zen-browser'`},
		},
		replies: []response{
			{output: `[{"id":"stale","app_id":"zen"}]`},
			{err: errors.New("unknown window")},
			{output: `[]`},
			{},
		},
	}

	err := activate(context.Background(), runner, "zen", []string{"zen-browser"})
	if err != nil {
		t.Fatalf("activate() error = %v", err)
	}
	runner.verify(t)
}

func TestActivateRejectsMalformedWindowJSON(t *testing.T) {
	runner := &fakeRunner{
		t:       t,
		want:    [][]string{{"windows", "--json"}},
		replies: []response{{output: `{not-json}`}},
	}

	err := activate(context.Background(), runner, "zen", []string{"zen-browser"})
	if err == nil || !strings.Contains(err.Error(), "decode window list") {
		t.Fatalf("activate() error = %v, want decode failure", err)
	}
	runner.verify(t)
}

func TestFormatShellCommandPreservesArguments(t *testing.T) {
	command := []string{
		"printf",
		"<%s>\n",
		"",
		"scratch pad",
		"it's literal",
		"$HOME; echo changed",
		"line\nbreak",
	}

	output, err := exec.Command("/bin/sh", "-c", formatShellCommand(command)).Output()
	if err != nil {
		t.Fatalf("execute formatted command: %v", err)
	}

	want := "<>\n<scratch pad>\n<it's literal>\n<$HOME; echo changed>\n<line\nbreak>\n"
	if got := string(output); got != want {
		t.Fatalf("formatted command output = %q, want %q", got, want)
	}
}

func TestParseOptions(t *testing.T) {
	var output strings.Builder
	opts, err := parseOptions([]string{
		"--app-id", "com.mitchellh.ghostty",
		"--umbriel", "/opt/umbriel",
		"--", "ghostty", "--class", "quick-term",
	}, &output)
	if err != nil {
		t.Fatalf("parseOptions() error = %v", err)
	}

	if opts.appID != "com.mitchellh.ghostty" || opts.umbrielPath != "/opt/umbriel" {
		t.Fatalf("parseOptions() = %#v", opts)
	}
	wantCommand := []string{"ghostty", "--class", "quick-term"}
	if !reflect.DeepEqual(opts.command, wantCommand) {
		t.Fatalf("command = %v, want %v", opts.command, wantCommand)
	}
}

func TestExecuteReportsUsageErrors(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := execute(context.Background(), []string{"--app-id", "zen"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("execute() = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "a launch command is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteWritesHelpToStandardOutput(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := execute(context.Background(), []string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("execute() = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: umbriel-raise") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
