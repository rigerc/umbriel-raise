package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type appChoice struct {
	AppID     string
	Title     string
	Workspace string
	Count     int
	Active    bool
}

func discoverApps(ctx context.Context, r runner) ([]appChoice, error) {
	windows, err := listWindows(ctx, r)
	if err != nil {
		return nil, err
	}

	byAppID := make(map[string]appChoice, len(windows))
	for _, candidate := range windows {
		if candidate.AppID == "" {
			continue
		}

		choice, found := byAppID[candidate.AppID]
		if !found {
			choice = appChoice{
				AppID:     candidate.AppID,
				Title:     candidate.Title,
				Workspace: candidate.Workspace,
			}
		}
		choice.Count++
		choice.Active = choice.Active || candidate.Active
		byAppID[candidate.AppID] = choice
	}

	apps := make([]appChoice, 0, len(byAppID))
	for _, choice := range byAppID {
		apps = append(apps, choice)
	}
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].AppID < apps[j].AppID
	})
	return apps, nil
}

func (a appChoice) Label() string {
	appID := sanitizeTerminalText(a.AppID)
	title := sanitizeTerminalText(a.Title)
	workspace := sanitizeTerminalText(a.Workspace)
	active := ""
	if a.Active {
		active = " • active"
	}

	detail := title
	if workspace != "" {
		detail = strings.TrimSpace(detail + " • " + workspace)
	}
	if detail != "" {
		detail = " — " + detail
	}

	return fmt.Sprintf("%s%s • %d window(s)%s", appID, detail, a.Count, active)
}

func sanitizeTerminalText(value string) string {
	return strings.Join(strings.FieldsFunc(value, func(r rune) bool {
		return unicode.IsControl(r) || unicode.In(r, unicode.Cf)
	}), " ")
}
