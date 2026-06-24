---
name: gooctl
description: Use gooctl to read and manage Gmail and Google Calendar from the terminal — search/read/send mail, view threads, list/create/update/delete calendar events. Verify auth first and use command-specific --help for exact flags.
---

# gooctl Agent Guide

Use this skill when the user wants to inspect or modify Gmail or Google Calendar through `gooctl`. The CLI covers two domains under one auth: `mail` and `calendar` (alias `cal`).

## Quick Rules

- Always verify auth before substantive work: `gooctl auth status`.
- Output is human-formatted text — there is **no `--json` flag**. Capture stdout and parse with shell tools (grep/awk) only when necessary; prefer narrow queries over post-filtering.
- Before write/delete actions (send mail, create/update/delete events), inspect current state first (`mail read`, `calendar show`) and confirm with the user when the action is destructive or externally visible.
- Use command-specific help for exact flags: `gooctl <command> <subcommand> --help`.
- Default limits are small (mail: 10, calendar varies). Pass `-n` / `--limit` for broader results.

## Auth + Credential Behavior

- One-time setup: `gooctl auth setup` (creates personal Google OAuth app — only needed once per machine).
- Login: `gooctl auth login` (browser flow).
- Status / logout: `gooctl auth status`, `gooctl auth logout`.
- Credentials live in `~/.config/gooctl/` (token.json + client config). Treat as sensitive.

## Command Map

- Auth: `gooctl auth {setup,login,status,logout}`
- Mail: `gooctl mail {search,unread,read,thread,send,reply,labels}`
- Calendar: `gooctl calendar {today,week,upcoming,search,show,create,update,delete,list}` (alias `cal`)

## Mail — Read Patterns

Mail search uses **Gmail query syntax** (same operators as the Gmail web UI). Be specific to keep results within the default limit.

```bash
# Unread inbox (default 10; raise with -n)
gooctl mail unread -n 50

# Search by sender / subject / date
gooctl mail search "from:someone@example.com"
gooctl mail search "subject:invoice"
gooctl mail search "is:unread after:2026/01/01"
gooctl mail search "from:@example.com -in:sent" -n 25

# Read full body / full thread
gooctl mail read <message-id>
gooctl mail thread <thread-id>

# Inventory labels (useful for label-scoped searches)
gooctl mail labels
```

### Superhuman label searches

If the user has Superhuman, it syncs AI labels into Gmail that gooctl can query:

```bash
gooctl mail search 'label:[Superhuman]/AI/Respond'
gooctl mail search 'is:inbox -label:[Superhuman]/AI/Pitch -label:[Superhuman]/AI/Marketing'
```

## Mail — Write Patterns

Sending and replying are externally visible — confirm the recipient, subject, and body with the user before invoking unless they have explicitly authorized the send.

```bash
gooctl mail send --to "x@y.com" --subject "Hi" --body "Hello!"
gooctl mail reply <message-id> --body "Thanks — confirmed."
```

For multi-line bodies, pass the body via a shell-quoted string or a heredoc-built variable; gooctl takes a single `--body` string.

## Calendar — Read Patterns

```bash
# Time-window views (limits: today=20, week=50, upcoming=10)
gooctl calendar today
gooctl cal week
gooctl cal upcoming -n 25

# Search across past/future window (default ±30 days; widen with --days)
gooctl cal search "standup"
gooctl cal search "Pete" --days 90 -n 25

# Drill into one event
gooctl cal show <event-id>

# Calendars the account has access to
gooctl cal list
```

## Calendar — Write Patterns

Times accept **RFC3339** or `YYYY-MM-DDTHH:MM:SS` (local). All-day events use `--date YYYY-MM-DD --all-day`.

```bash
# Timed event
gooctl cal create --title "Strategy sync" \
  --start "2026-05-04T10:00:00" --end "2026-05-04T11:00:00" \
  --location "Zoom" --description "Q2 review" \
  --attendees "pete@example.com,advisor@example.com"

# All-day event
gooctl cal create --title "Quiet day" --date "2026-05-04" --all-day

# Update fields or shift the event
gooctl cal update <event-id> --title "Strategy sync (rescheduled)"
gooctl cal update <event-id> --start "..." --end "..."
gooctl cal update <event-id> --move-days 1   # +1 day; negative = past

# Delete (always confirm with the user first)
gooctl cal delete <event-id>
```

Confirm before `delete` and before `update` on events with attendees — both trigger Google Calendar notifications to invitees.

## Core Workflow

1. Confirm auth (`gooctl auth status`).
2. Discover the target — search/list with narrow filters to find the right ID.
3. Read the exact target (`mail read`, `mail thread`, `cal show`) before mutating.
4. For writes, confirm with the user when the action sends mail, invites attendees, or deletes data.
5. Apply the change, then re-read the target and report concrete outcome (message ID, event link, etc.).

## High-Impact Gotchas

- **No `--json`.** Don't fabricate structured output; quote what gooctl actually printed.
- **Default limits hide results.** Mail commands default to 10; calendar `search` and `upcoming` default to 10. Always raise `-n` when the user asks "all" or "everything."
- **Calendar search window** is ±30 days by default. Use `--days` for older/further events.
- **Time zones.** Local-style `YYYY-MM-DDTHH:MM:SS` is interpreted in the system local TZ; use full RFC3339 (`2026-05-04T10:00:00-04:00`) when the user specifies a different zone.
- **All-day vs timed.** `--all-day` requires `--date` (not `--start`/`--end`).
- **Sends/invites are real.** `mail send`, `mail reply`, and any `cal create`/`update` with `--attendees` notify recipients — do not invoke speculatively.

## Troubleshooting

- `Not authenticated` / token error → `gooctl auth login`, then `gooctl auth status`.
- First-time machine with no client config → `gooctl auth setup` (interactive, ~2 min).
- `unknown command` / `unknown flag` → binary may be old; re-run `go install github.com/dorkitude/gooctl@latest` and re-check `gooctl --help`.
- Empty search results → broaden the Gmail query, raise `-n`, or for calendar widen `--days`.
- Validation errors → run the exact subcommand help and retry:
  - `gooctl mail send --help`
  - `gooctl calendar create --help`
  - `gooctl calendar update --help`

## Minimal Discovery Commands

```bash
gooctl --help
gooctl auth --help
gooctl mail --help
gooctl calendar --help
```
