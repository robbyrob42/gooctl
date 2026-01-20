# CLAUDE.md

This file provides guidance to Claude Code when working with gooctl.

## Project Overview

gooctl is a CLI for Google Workspace (Gmail & Calendar) built in Go with lipgloss for beautiful terminal UI.

## Build & Run

```bash
go build -o gooctl .      # Build the binary
go run .                   # Run directly
go install .               # Install to $GOPATH/bin
```

## Project Structure

- `main.go` - Entry point
- `cmd/` - Cobra CLI commands (root, auth, mail, calendar)
- `internal/auth/` - OAuth2 flow and token management
- `internal/gmail/` - Gmail API client
- `internal/calendar/` - Google Calendar API client
- `internal/ui/` - Lipgloss styling

## Key Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- Google APIs: gmail, calendar, oauth2

## Config

Credentials stored in `~/.config/gooctl/`
