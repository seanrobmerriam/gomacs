package main

import "fmt"

const lineNumDigits = 4
const gutterWidth = lineNumDigits + 1 // digits + 1 space separator

// layoutSizes computes the widths and content height for the two-panel layout
func layoutSizes(width, height int) (editorWidth, explorerWidth, contentHeight int) {
	explorerWidth = width / 4
	if explorerWidth < 15 {
		explorerWidth = 15
	}
	if explorerWidth > width-10 {
		explorerWidth = width - 10
	}
	if explorerWidth < 0 {
		explorerWidth = 0
	}
	editorWidth = width - explorerWidth - 1 // -1 for divider
	if editorWidth < 0 {
		editorWidth = 0
	}
	contentHeight = height - 1 // -1 for status bar
	if contentHeight < 0 {
		contentHeight = 0
	}
	return
}

// View renders the entire model to a screen buffer
func View(model Model, screen *Screen) {
	editorW, explorerW, contentH := layoutSizes(model.Width, model.Height)

	// Editor panel (left)
	renderEditor(screen, model.Editor, 0, 0, editorW, contentH, model.Focus == EditorPanel)

	// Vertical divider
	dividerX := editorW
	dividerStyle := Style{FG: ColorWhite}
	for y := 0; y < contentH; y++ {
		screen.Set(dividerX, y, '│', dividerStyle)
	}

	// Explorer panel (right)
	renderExplorer(screen, model.Explorer, dividerX+1, 0, explorerW, contentH, model.Focus == ExplorerPanel)

	// Status bar (bottom)
	renderStatusBar(screen, model, model.Height-1)
}

func renderEditor(screen *Screen, editor EditorModel, x0, y0, width, height int, focused bool) {
	if width <= gutterWidth || height <= 0 {
		return
	}
	textX0 := x0 + gutterWidth
	textWidth := width - gutterWidth

	gutterStyle := Style{FG: ColorYellow}
	textStyle := DefaultStyle

	for row := 0; row < height; row++ {
		lineIdx := editor.ScrollY + row
		if lineIdx >= len(editor.Lines) {
			// Show tilde for lines beyond the file
			screen.Set(x0, y0+row, '~', Style{FG: ColorBlue})
			continue
		}

		// Line number
		numStr := fmt.Sprintf("%*d", lineNumDigits, lineIdx+1)
		screen.SetString(x0, y0+row, numStr, gutterStyle)

		// Text content
		line := []rune(editor.Lines[lineIdx])
		for col := 0; col < textWidth; col++ {
			charIdx := editor.ScrollX + col
			if charIdx < len(line) {
				screen.Set(textX0+col, y0+row, line[charIdx], textStyle)
			}
		}
	}
}

func renderExplorer(screen *Screen, explorer ExplorerModel, x0, y0, width, height int, focused bool) {
	if width <= 0 || height <= 0 {
		return
	}

	// Header
	headerStyle := Style{Bold: true, FG: ColorCyan}
	screen.SetString(x0, y0, " Files", headerStyle)

	for row := 1; row < height; row++ {
		entryIdx := explorer.ScrollY + row - 1
		if entryIdx >= len(explorer.Entries) {
			break
		}
		entry := explorer.Entries[entryIdx]

		// Determine style
		style := DefaultStyle
		isSelected := entryIdx == explorer.Selected && focused
		if entry.IsDir {
			style.FG = ColorBlue
			style.Bold = true
		}
		if isSelected {
			style.Reverse = true
		}

		// Build display string with indentation
		indent := entry.Depth * 2
		var prefix string
		if entry.IsDir {
			if entry.Open {
				prefix = "▾ "
			} else {
				prefix = "▸ "
			}
		} else {
			prefix = "  "
		}

		display := ""
		for i := 0; i < indent; i++ {
			display += " "
		}
		display += prefix + entry.Name

		// Fill background for selected row
		if isSelected {
			for col := 0; col < width; col++ {
				screen.Set(x0+col, y0+row, ' ', style)
			}
		}

		// Write the entry text
		runes := []rune(display)
		if len(runes) > width {
			runes = runes[:width]
		}
		for col, ch := range runes {
			if x0+col < screen.Width {
				screen.Set(x0+col, y0+row, ch, style)
			}
		}
	}
}

func renderStatusBar(screen *Screen, model Model, y int) {
	style := Style{Reverse: true}

	// Fill the status bar background
	for x := 0; x < model.Width; x++ {
		screen.Set(x, y, ' ', style)
	}

	// Left: filename and modified indicator
	filename := model.Editor.Filename
	if filename == "" {
		filename = "[scratch]"
	}
	mod := ""
	if model.Editor.Modified {
		mod = " [modified]"
	}
	left := fmt.Sprintf(" %s%s", filename, mod)
	screen.SetString(0, y, left, style)

	// Right: cursor position and status message
	right := fmt.Sprintf("L%d:C%d  %s ", model.Editor.CursorY+1, model.Editor.CursorX+1, model.Status)
	rightRunes := []rune(right)
	leftRunes := []rune(left)
	rightX := model.Width - len(rightRunes)
	if rightX < len(leftRunes)+1 {
		rightX = len(leftRunes) + 1
	}
	screen.SetString(rightX, y, right, style)
}

// CursorPosition returns the screen coordinates for the editor cursor
func CursorPosition(model Model) (int, int) {
	return gutterWidth + model.Editor.CursorX - model.Editor.ScrollX,
		model.Editor.CursorY - model.Editor.ScrollY
}
