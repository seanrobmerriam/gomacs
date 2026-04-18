# gomacs

A native TUI text editor written in Go, built on the [Elm architecture](https://guide.elm-lang.org/architecture/) with zero external dependencies.

Runs on Linux, macOS, and FreeBSD.

## Features

- **Two-panel layout** — editor on the left, file explorer on the right
- **Emacs-style keybindings** — `C-x C-s` save, `C-x C-f` open, `C-a/e/n/p/f/b` movement
- **Multi-file buffers** — open multiple files simultaneously, cycle with `C-x b`, close with `C-x k`
- **File explorer** — browse directories, expand/collapse folders, open files with Enter or mouse click
- **Syntax highlighting** — Go, C/C++, Python, JS/TS, Shell; hand-written tokenizer, zero allocations per keypress
- **Incremental search** — `C-s` forward search with match highlight, `C-g`/`Esc` to cancel
- **Undo** — `C-/` or `C-_` steps back through edit history (up to 1000 levels)
- **Mouse support** — click to position cursor, click explorer entries to open/toggle, scroll wheel to scroll
- **Status line polish** — shows current language and line ending (`LF`/`CRLF`)
- **Double-buffered rendering** — diff-based ANSI updates, no flicker
- **Live terminal resize** — `SIGWINCH` signal handling, redraws immediately
- **Raw terminal** — via `syscall` + `ioctl`, no external libraries

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `Tab` | Switch focus between editor and explorer |
| `C-x C-c` | Quit |
| `C-x C-s` | Save current file |
| `C-x C-f` | Switch to file explorer (then Enter to open) |
| `C-x b` | Cycle to next open buffer |
| `C-x k` | Kill (close) current buffer |

### Editor

| Key | Action |
|-----|--------|
| `C-a` / `C-e` | Beginning / end of line |
| `C-f` / `C-b` | Forward / backward character |
| `C-n` / `C-p` | Next / previous line |
| `C-d` | Delete character forward |
| `C-k` | Kill to end of line |
| `C-s` | Incremental forward search (type to refine, `C-s` next match, `Enter` confirm, `C-g`/`Esc` cancel) |
| `C-/` or `C-_` | Undo |
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

### Mouse

| Input | Action |
|-----|--------|
| Left click in editor | Move cursor to clicked position |
| Left click file entry in explorer | Open file |
| Left click directory entry in explorer | Toggle expand/collapse |
| Wheel up/down | Scroll focused panel |

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
`fmt`, `os`, `os/signal`, `syscall`, `unicode/utf8`, `path/filepath`, `sort`, `strings`, `unsafe`, `bytes`.

## Run

```bash
./gomacs              # start with empty scratch buffer
./gomacs file.go      # open an existing file
```

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point, Elm runtime loop, SIGWINCH handling |
| `model.go` | Model, Msg, Cmd, Update function, search logic |
| `view.go` | View function, layout, cursor position |
| `screen.go` | Cell grid, ANSI diff renderer |
| `editor.go` | Editor sub-model (buffer, cursor, scroll, undo) |
| `explorer.go` | File explorer sub-model (tree, selection) |
| `highlight.go` | Syntax tokenizer, language detection |
| `input.go` | Input parser (keyboard + mouse escape sequences) |
| `terminal.go` | Raw mode, terminal size via `ioctl` |
| `terminal_linux.go` | Linux `ioctl` constants |
| `terminal_bsd.go` | macOS / FreeBSD `ioctl` constants |
