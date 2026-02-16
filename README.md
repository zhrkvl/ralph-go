# Minimal Ralph in Go

Lightweight Go rewrite of [Ralph](https://github.com/snarktank/ralph) with a built-in TUI. Drives Claude or Amp through user stories defined in a `prd.json`, showing real-time agent output, story progress, and session history.

![Go](https://img.shields.io/badge/Go-1.25-blue)

## Install

```bash
go install github.com/zhrkvl/ralph-go@latest
```

Or build from source:

```bash
git clone https://github.com/zhrkvl/ralph-go.git
cd ralph-go && go build -o ralph .
```

## Usage

```bash
# Run from a project that has scripts/ralph/prd.json
cd your-project
ralph --tool claude --max-iterations 10

# Or point to the ralph directory explicitly
ralph --ralph-dir ./scripts/ralph --tool claude
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--tool` | `amp` | Agent: `claude` or `amp` |
| `--max-iterations` | `10` | Max agent iterations before stopping |
| `--ralph-dir` | auto | Directory containing `prd.json` and `CLAUDE.md` |
| `--project-dir` | CWD | Working directory for the agent |

Ralph auto-discovers `--ralph-dir` by checking: `RALPH_DIR` env var → `./scripts/ralph/` → CWD.

## TUI

Three views, cycle with `Tab`:

**Dashboard** — live agent output, current story, progress bar
**Stories** — browse all user stories from prd.json
**History** — view archived sessions

### Keys

| Key | Action |
|-----|--------|
| `Tab` | Cycle views |
| `p` | Pause/resume agent |
| `s` | Skip current iteration |
| `q` | Quit (confirms if agent running) |
| `↑↓` / `jk` | Scroll / navigate |
| `Enter` | View story or archive details |
| `Esc` | Back |

## PRD Format

Ralph expects a `prd.json` in the ralph directory:

```json
{
  "name": "MyProject",
  "description": "...",
  "branchName": "ralph/my-feature",
  "userStories": [
    {
      "id": "US-001",
      "title": "Implement feature X",
      "description": "As a user, I need...",
      "acceptanceCriteria": ["criterion 1", "criterion 2"],
      "priority": 1,
      "passes": false
    }
  ]
}
```

Each iteration, the agent picks the highest-priority story where `passes: false`, implements it, and marks it done. When all stories pass, the agent emits `<promise>COMPLETE</promise>` and Ralph stops.

## How It Works

1. Load `prd.json` and check for branch changes (archives previous run if branch differs)
2. Start agent loop: invoke `claude --print --output-format stream-json` (or `amp`)
3. Stream output to the TUI in real time (token-by-token for Claude)
4. On completion signal or max iterations, stop
5. Sleep 2s between iterations

Agent instructions are read from `CLAUDE.md` (for Claude) or `prompt.md` (for Amp) in the ralph directory.
