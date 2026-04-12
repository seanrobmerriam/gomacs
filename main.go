package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Get working directory for the file explorer
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Enable terminal raw mode
	fd := int(os.Stdin.Fd())
	orig, err := enableRawMode(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling raw mode: %v\n", err)
		os.Exit(1)
	}
	defer disableRawMode(fd, orig)

	// Use alternate screen buffer to preserve the user's scrollback
	enterAltScreen(os.Stdout)
	defer exitAltScreen(os.Stdout)
	defer fmt.Fprint(os.Stdout, "\x1b[?25h") // ensure cursor is visible on exit

	// Get terminal dimensions
	width, height, err := getTerminalSize(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting terminal size: %v\n", err)
		os.Exit(1)
	}

	// Initialize application state
	model := InitModel(width, height, dir)

	// Open file from command line argument, if provided
	if len(os.Args) > 1 {
		filename := os.Args[1]
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(dir, filename)
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			model.Status = fmt.Sprintf("Error opening %s: %v", os.Args[1], err)
		} else {
			// Process the open through the Elm Update cycle
			model, _ = Update(model, FileOpenedMsg{filename, string(data)})
		}
	}

	ClearScreen(os.Stdout)

	var prevScreen *Screen

	// Main loop: View → Render → ReadInput → Update
	for !model.Quit {
		// View: model → screen buffer
		screen := NewScreen(model.Width, model.Height)
		View(model, screen)

		// Render: write screen to terminal
		screen.Render(os.Stdout, prevScreen)

		// Show cursor at editor position (or hide if explorer focused)
		if model.Focus == EditorPanel {
			cx, cy := CursorPosition(model)
			SetCursor(os.Stdout, cx, cy)
		}

		prevScreen = screen

		// Read input → message
		key := ReadKey(os.Stdin)
		if key.Key == KeyNone {
			continue
		}

		// Update: (model, msg) → (model, cmd)
		var cmd Cmd
		model, cmd = Update(model, KeyMsg{key})

		// Execute command chain (Elm runtime)
		for cmd != nil {
			msg := cmd()
			model, cmd = Update(model, msg)
		}

		// Check for terminal resize
		if newW, newH, err := getTerminalSize(fd); err == nil {
			if newW != model.Width || newH != model.Height {
				model.Width = newW
				model.Height = newH
				prevScreen = nil // force full redraw
			}
		}
	}
}
