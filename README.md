# gomacs

A native TUI text editor written in Go, built on the [Elm architecture](https://guide.elm-lang.org/architecture/) with zero external dependencies.

Runs on Linux, macOS, and FreeBSD.

## Features

- **Two-panel layout** — editor on the left, file explorer on the right
- **Emacs-style keybindings** — `C-x C-s` save, `C-x C-f` open, `C-a/e/n/p/f/b` movement
- **File explorer** — browse directories, expand/collapse folders, open files with Enter
- **Double-buffered rendering** — diff-based ANSI updates, no flicker
- **Raw terminal** — via `syscall` + `ioctl`, no external libraries

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `Tab` | Switch focus between editor and explorer |
| `C-x C-c` | Quit |
| `C-x C-s` | Save current file |
| `C-x C-f` | Switch to file explorer (then Enter to open) |

### Editor

| Key | Action |
|-----|--------|
| `C-a` / `C-e` | Beginning / end of line |
| `C-f` / `C-b` | Forward / backward character |
| `C-n` / `C-p` | Next / previous line |
| `C-d` | Delete character forward |
| `C-k` | Kill to end of line |
| `Backspace` | Delete character backward |
| `Enter` | Insert newline |
| Arrow keys | Cursor movement |
| `PgUp` / `PgDn` | Page up / down |

### File Explorer

| Key | Action |
|-----|--------|
| `Up` / `Down` or `C-p` / `C-n` | Move selection |
| `Enter` | Open file (or toggle directory) |
| `Right` | Expand directory |
| `Left` | Collapse directory |

## Architecture

```
┌────────────────────────────────────────┐
│                 main loop              │
│  ┌──────┐    ┌──────┐    ┌────────┐    │
│  │ Init │───▶│ View │───▶│ Render │──▶ │
│  └──────┘    └──┬───┘    └────────┘    │
│                 ▲                      │
│                 │     ┌────────┐       │
│                 └─────│ Update │◀──────│
│                       └────────┘       │
└────────────────────────────────────────┘
```

- **Model** — immutable application state (editor buffer, cursor, file tree, focus)
- **View** — pure function: `(Model, Screen) → ()` renders to a cell grid
- **Update** — pure function: `(Model, Msg) → (Model, Cmd)` state transitions
- **Cmd** — side effects (file I/O) that produce messages back into the loop

## Build

```bash
go build -o gomacs .
```

No external dependencies. The only imports are the Go standard library packages:
`fmt`, `os`, `syscall`, `unicode/utf8`, `path/filepath`, `sort`, `strings`, `unsafe`, `bytes`.

## Run

```bash
./gomacs              # start with empty scratch buffer
./gomacs file.go      # open an existing file
```

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point, Elm runtime loop |
| `model.go` | Model, Msg, Cmd, Update function |
| `view.go` | View function, layout, cursor position |
| `screen.go` | Cell grid, ANSI diff renderer |
| `editor.go` | Editor sub-model (buffer, cursor, scroll) |
| `explorer.go` | File explorer sub-model (tree, selection) |
| `input.go` | Key parser (UTF-8, escape sequences, ctrl keys) |
| `terminal.go` | Raw mode, terminal size via `ioctl` |
| `terminal_linux.go` | Linux `ioctl` constants |
| `terminal_bsd.go` | macOS / FreeBSD `ioctl` constants |
