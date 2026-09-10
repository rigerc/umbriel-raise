package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectCycleTarget(t *testing.T) {
	tests := []struct {
		name    string
		windows []window
		wantID  string
		found   bool
	}{
		{name: "no windows", found: false},
		{
			name:    "no active window",
			windows: []window{{ID: "one", AppID: "kitty"}, {ID: "two", AppID: "kitty"}},
			found:   false,
		},
		{
			name: "workspace local focus does not select the app",
			windows: []window{
				{ID: "one", AppID: "kitty", Focused: true},
				{ID: "two", AppID: "kitty", Focused: true},
			},
			found: false,
		},
		{
			name:    "active window without an app ID",
			windows: []window{{ID: "one", Active: true}, {ID: "two"}},
			found:   false,
		},
		{
			name:    "focused app has a single window",
			windows: []window{{ID: "one", AppID: "kitty", Active: true}, {ID: "two", AppID: "emacs"}},
			found:   false,
		},
		{
			name: "two windows rotate to the other match",
			windows: []window{
				{ID: "one", AppID: "kitty", Active: true},
				{ID: "two", AppID: "kitty"},
			},
			wantID: "two",
			found:  true,
		},
		{
			name: "three windows rotate to the least recent match",
			windows: []window{
				{ID: "one", AppID: "kitty", Active: true},
				{ID: "two", AppID: "kitty"},
				{ID: "three", AppID: "kitty"},
			},
			wantID: "three",
			found:  true,
		},
		{
			name: "other applications stay out of the rotation",
			windows: []window{
				{ID: "one", AppID: "kitty", Active: true},
				{ID: "browser", AppID: "app.zen_browser.zen"},
				{ID: "two", AppID: "kitty"},
				{ID: "editor", AppID: "emacs"},
			},
			wantID: "two",
			found:  true,
		},
		{
			name: "app ID matching stays exact and case sensitive",
			windows: []window{
				{ID: "one", AppID: "kitty", Active: true},
				{ID: "two", AppID: "Kitty"},
				{ID: "three", AppID: "kitty-extra"},
			},
			found: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, found := selectCycleTarget(test.windows)
			if found != test.found || got.ID != test.wantID {
				t.Fatalf("selectCycleTarget() = (%q, %v), want (%q, %v)", got.ID, found, test.wantID, test.found)
			}
		})
	}
}

func TestCycleFocusedRotatesToLeastRecentMatch(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "window-focus-warp:second"},
		},
		replies: []response{
			{output: `[
				{"id":"first","app_id":"kitty","active":true,"focused":true},
				{"id":"browser","app_id":"app.zen_browser.zen","focused":false},
				{"id":"second","app_id":"kitty","focused":false}
			]`},
			{},
		},
	}

	if err := cycleFocused(context.Background(), runner); err != nil {
		t.Fatalf("cycleFocused() error = %v", err)
	}
	runner.verify(t)
}

func TestCycleFocusedIsQuietWhenFocusedAppHasOneWindow(t *testing.T) {
	runner := &fakeRunner{
		t:    t,
		want: [][]string{{"windows", "--json"}},
		replies: []response{
			{output: `[
				{"id":"only","app_id":"kitty","active":true},
				{"id":"editor","app_id":"emacs"}
			]`},
		},
	}

	if err := cycleFocused(context.Background(), runner); err != nil {
		t.Fatalf("cycleFocused() error = %v", err)
	}
	runner.verify(t)
}

func TestCycleFocusedIsQuietWhenNothingIsActive(t *testing.T) {
	runner := &fakeRunner{
		t:    t,
		want: [][]string{{"windows", "--json"}},
		replies: []response{
			{output: `[{"id":"one","app_id":"kitty","focused":true}]`},
		},
	}

	if err := cycleFocused(context.Background(), runner); err != nil {
		t.Fatalf("cycleFocused() error = %v", err)
	}
	runner.verify(t)
}

func TestCycleFocusedRefreshesAfterStaleWindowID(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "window-focus-warp:stale"},
			{"windows", "--json"},
			{"msg", "window-focus-warp:replacement"},
		},
		replies: []response{
			{output: `[
				{"id":"live","app_id":"kitty","active":true},
				{"id":"stale","app_id":"kitty"}
			]`},
			{err: errors.New("unknown window")},
			{output: `[
				{"id":"live","app_id":"kitty","active":true},
				{"id":"replacement","app_id":"kitty"}
			]`},
			{},
		},
	}

	if err := cycleFocused(context.Background(), runner); err != nil {
		t.Fatalf("cycleFocused() error = %v", err)
	}
	runner.verify(t)
}

func TestCycleFocusedStopsWhenTargetDisappears(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "window-focus-warp:stale"},
			{"windows", "--json"},
		},
		replies: []response{
			{output: `[
				{"id":"live","app_id":"kitty","active":true},
				{"id":"stale","app_id":"kitty"}
			]`},
			{err: errors.New("unknown window")},
			{output: `[{"id":"live","app_id":"kitty","active":true}]`},
		},
	}

	if err := cycleFocused(context.Background(), runner); err != nil {
		t.Fatalf("cycleFocused() error = %v", err)
	}
	runner.verify(t)
}

func TestCycleFocusedReportsFocusFailure(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "window-focus-warp:second"},
			{"windows", "--json"},
			{"msg", "window-focus-warp:second"},
		},
		replies: []response{
			{output: `[{"id":"first","app_id":"kitty","active":true},{"id":"second","app_id":"kitty"}]`},
			{err: errors.New("ipc down")},
			{output: `[{"id":"first","app_id":"kitty","active":true},{"id":"second","app_id":"kitty"}]`},
			{err: errors.New("ipc down")},
		},
	}

	err := cycleFocused(context.Background(), runner)
	if err == nil || !strings.Contains(err.Error(), "focus failed after refresh") {
		t.Fatalf("cycleFocused() error = %v, want focus failure after refresh", err)
	}
	runner.verify(t)
}

func TestCycleFocusedRejectsMalformedWindowJSON(t *testing.T) {
	runner := &fakeRunner{
		t:       t,
		want:    [][]string{{"windows", "--json"}},
		replies: []response{{output: `{not-json}`}},
	}

	err := cycleFocused(context.Background(), runner)
	if err == nil || !strings.Contains(err.Error(), "decode window list") {
		t.Fatalf("cycleFocused() error = %v, want decode failure", err)
	}
	runner.verify(t)
}

func TestParseCycleOptions(t *testing.T) {
	var output strings.Builder

	opts, err := parseCycleOptions([]string{"--umbriel", "/opt/umbriel"}, &output)
	if err != nil {
		t.Fatalf("parseCycleOptions() error = %v", err)
	}
	if opts.umbrielPath != "/opt/umbriel" {
		t.Fatalf("parseCycleOptions() = %#v", opts)
	}

	defaults, err := parseCycleOptions(nil, &output)
	if err != nil {
		t.Fatalf("parseCycleOptions() error = %v", err)
	}
	if defaults.umbrielPath != "umbriel" {
		t.Fatalf("default umbriel path = %q", defaults.umbrielPath)
	}

	if _, err := parseCycleOptions([]string{"extra"}, &output); err == nil {
		t.Fatal("parseCycleOptions() accepted a positional argument")
	}
}

func TestExecuteRoutesCycle(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := execute(context.Background(), []string{"cycle", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("execute() = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage: umbriel-raise cycle") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestExecuteCycleReportsUsageErrors(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := execute(context.Background(), []string{"cycle", "extra"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("execute() = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unexpected argument "extra"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

// fakeUmbriel writes an executable stub that answers every invocation with the
// given stdout, standing in for the Umbriel CLI so the command layer can be
// exercised without a compositor.
func fakeUmbriel(t *testing.T, output string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "umbriel")
	script := "#!/bin/sh\ncat <<'FIXTURE'\n" + output + "\nFIXTURE\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake umbriel: %v", err)
	}
	return path
}

func TestRunCycleCLISucceedsWhenThereIsNothingToRotate(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := runCycleCLI(
		context.Background(),
		[]string{"--umbriel", fakeUmbriel(t, `[{"id":"one","app_id":"kitty","active":true}]`)},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("runCycleCLI() = %d, want 0", code)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q, want both empty", stdout.String(), stderr.String())
	}
}

func TestRunCycleCLIReportsUmbrielFailures(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	missing := filepath.Join(t.TempDir(), "does-not-exist")
	code := runCycleCLI(context.Background(), []string{"--umbriel", missing}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("runCycleCLI() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "umbriel-raise cycle: list windows") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestExecuteCycleReportsUnknownFlags(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := execute(context.Background(), []string{"cycle", "--bogus"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("execute() = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr = %q, want the flag package diagnostic forwarded", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Usage: umbriel-raise cycle") {
		t.Fatalf("stderr = %q, want usage text", stderr.String())
	}
}

func TestCycleFocusedReportsRefreshFailure(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		want: [][]string{
			{"windows", "--json"},
			{"msg", "window-focus-warp:second"},
			{"windows", "--json"},
		},
		replies: []response{
			{output: `[{"id":"first","app_id":"kitty","active":true},{"id":"second","app_id":"kitty"}]`},
			{err: errors.New("unknown window")},
			{err: errors.New("ipc closed")},
		},
	}

	err := cycleFocused(context.Background(), runner)
	if err == nil || !strings.Contains(err.Error(), "refresh after focus failure") {
		t.Fatalf("cycleFocused() error = %v, want refresh failure", err)
	}
	runner.verify(t)
}
