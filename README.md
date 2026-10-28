# linear-term

A keyboard-first terminal Kanban dashboard for Linear's **Delivery** board view, built in Go with [Charm's Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Multi-Column Kanban Delivery View**: Renders `Backlog`, `Todo`, `In Progress`, `In Review`, `Rolling Out`, `Done (Past week)`, and `Duplicate (Past week)` columns with truecolor Linear dark styling.
- **Smart Empty-Column Collapse (`e`)**: Automatically collapses `0`-count columns (`Backlog 0`, `Todo 0`, `Duplicate 0`) so active delivery columns fill your terminal width, with a one-key toggle (`e`) to expand all 7 columns.
- **Rich Issue Cards**:
  - Priority badges (`!` Urgent orange pill, `▂▄▆` High, `▂▄░` Medium, `▂░░` Low, `···` None)
  - Project badges (`❖ Storage engine`, `❖ Edge runtime`, `❖ Regions & failover`, `❖ Pipeline feeds`, `❖ Onboarding & Setup`, `❖ Query planner`)
  - Colored label dots (`● slow`, `● fast`, `● train`, `● Bug`, `● Improvement`, `● Feature`)
  - Linked GitHub PR badges (`⇅ 4 PRs`, `⇅ #8269`, `⇅ #3192`) and sub-issue completion (`✔ 3/3`)
- **Terminal Markdown Renderer**: Formats issue descriptions and comments in the detail modal (headings, nested bullets, task checkboxes, inline code, fenced code blocks, links with GitHub issue/PR `#number` extraction, bold/italic/strikethrough).
- **Live Polling & Visual Flash Highlights**: Polls Linear's GraphQL API on a configurable interval (default `15s`) and highlights newly updated or moved cards with an `⚡ UPDATED` / `⚡ MOVED` gold border.
- **Instant Demo Mode**: Comes pre-seeded with a sample Delivery board so you can test drive it immediately without an API key, or press `L` to simulate live incoming card updates.

## Installation

```bash
go install github.com/the2dl/linear-term@latest
```

Or build from source:

```bash
git clone https://github.com/the2dl/linear-term.git
cd linear-term
go build -o linear-term .
```

## Usage

```bash
# Run immediately (uses LINEAR_API_KEY or ~/.config/linear-term/token if set, otherwise opens in Demo mode)
./linear-term

# Run with your Linear Personal API Key
LINEAR_API_KEY="lin_api_..." ./linear-term

# Filter to a specific team key and custom poll interval (seconds)
./linear-term -team LIN -interval 10

# Force Demo mode with sample board data
./linear-term --demo
```

> **Tip**: Press **`A`** directly inside the TUI to paste and save your `LINEAR_API_KEY` to `~/.config/linear-term/token` (`0600`) and switch to Live API mode on the fly.

## Keyboard Shortcuts

| Key | Action |
| --- | --- |
| `h` / `l` or `←` / `→` | Move between workflow columns |
| `j` / `k` or `↓` / `↑` | Move between issue cards in a column (auto-scrolls column viewport) |
| `g` / `G` | Jump to top / bottom card in column |
| `Enter` / `Space` | Open full **Issue Detail Modal** (description, PR links, sub-issues, comments) |
| `s` | Open **Workflow State Picker** to move issue across columns (`1`–`7`) |
| `p` | Open **Priority Picker** to update issue priority (`1`–`5`) |
| `o` | Open selected issue in browser (`linear.app`) |
| `g` *(in detail modal)* | Open linked GitHub PR in browser |
| `y` | Copy issue identifier (e.g. `LIN-4759`) to clipboard (`wl-copy` / `xclip` / OSC52) |
| `b` | Copy issue git branch name to clipboard |
| `/` | Live filter/search across issue ID, title, project, label, or PR number |
| `1` / `2` / `3` / `4` or `Tab` | Switch views (`Active`, `Backlog`, `All issues`, `Delivery`) |
| `e` | Toggle hiding/showing empty (`0`-count) columns |
| `r` | Trigger immediate refresh from Linear API |
| `L` | Simulate a live incoming update / card transition |
| `A` | Configure or update `LINEAR_API_KEY` |
| `?` | Show keyboard help modal |
| `q` / `Ctrl+C` | Quit |

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
