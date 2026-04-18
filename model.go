package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Panel represents which panel has focus
type Panel int

const (
	EditorPanel Panel = iota
	ExplorerPanel
)

// --- Messages ---

// Msg is the interface for all messages in the Elm architecture
type Msg interface{}

// Cmd is a side-effect function that returns a Msg when executed
type Cmd func() Msg

// KeyMsg wraps a key event as a message
type KeyMsg struct{ KeyEvent }

// FileSavedMsg signals that a file was saved successfully
type FileSavedMsg struct{ Filename string }

// FileOpenedMsg signals that a file was loaded successfully
type FileOpenedMsg struct {
	Filename string
	Content  string
}

// ErrorMsg signals that a side effect failed
type ErrorMsg struct{ Err error }

// MouseMsg signals a mouse click or scroll event.
// Btn: 0=left click, 64=scroll up, 65=scroll down.
// X, Y are 0-based screen coordinates.
type MouseMsg struct {
	Btn int
	X   int
	Y   int
}

// --- Model ---

// Model is the complete application state
type Model struct {
	Buffers       []EditorModel
	BufIdx        int
	Explorer      ExplorerModel
	Focus         Panel
	Width         int
	Height        int
	Quit          bool
	Status        string
	Prefix        Key // for multi-key sequences (e.g., C-x)
	SearchMode    bool
	SearchQuery   string
	SearchOriginY int
	SearchOriginX int
}

// InitModel creates the initial application state
func InitModel(width, height int, dir string) Model {
	return Model{
		Buffers:  []EditorModel{NewEditorModel()},
		BufIdx:   0,
		Explorer: NewExplorerModel(dir),
		Width:    width,
		Height:   height,
		Status:   "C-x C-c quit | C-x C-s save | C-x C-f files | Tab switch",
	}
}

// --- Update ---

// Update is the top-level Elm update function.
// It takes the current model and a message, returns the new model and an optional command.
func Update(model Model, msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case KeyMsg:
		return updateKey(model, msg.KeyEvent)

	case FileSavedMsg:
		// Find the buffer with this filename and clear its modified flag
		for i := range model.Buffers {
			if model.Buffers[i].Filename == msg.Filename {
				model.Buffers[i].Modified = false
				break
			}
		}
		model.Status = fmt.Sprintf("Saved %s", filepath.Base(msg.Filename))
		return model, nil

	case FileOpenedMsg:
		lineEnding := detectLineEnding(msg.Content)
		normalized := normalizeLineEndings(msg.Content)
		lines := strings.Split(normalized, "\n")
		if len(lines) == 0 {
			lines = []string{""}
		}
		// If this file is already open in a buffer, switch to it
		for i, buf := range model.Buffers {
			if buf.Filename == msg.Filename {
				model.BufIdx = i
				model.Focus = EditorPanel
				model.Status = fmt.Sprintf("Switched to %s", filepath.Base(msg.Filename))
				return model, nil
			}
		}
		// Otherwise open as a new buffer
		newBuf := EditorModel{
			Lines:      lines,
			Filename:   msg.Filename,
			Lang:       DetectLanguage(msg.Filename),
			LineEnding: lineEnding,
		}
		model.Buffers = append(model.Buffers, newBuf)
		model.BufIdx = len(model.Buffers) - 1
		model.Focus = EditorPanel
		model.Status = fmt.Sprintf("Opened %s [%d/%d]", filepath.Base(msg.Filename), model.BufIdx+1, len(model.Buffers))
		return model, nil

	case ErrorMsg:
		model.Status = fmt.Sprintf("Error: %v", msg.Err)
		return model, nil

	case MouseMsg:
		return updateMouse(model, msg)
	}
	return model, nil
}

func updateMouse(model Model, msg MouseMsg) (Model, Cmd) {
	editorW, _, contentH := layoutSizes(model.Width, model.Height)
	dividerX := editorW
	explorerHeight := model.Height - 2 // -1 status bar, -1 explorer header

	switch msg.Btn {
	case 0: // left click
		if msg.X < dividerX {
			// Click in editor panel
			model.Focus = EditorPanel
			e := model.Buffers[model.BufIdx]
			textCol := msg.X - gutterWidth
			if textCol < 0 {
				textCol = 0
			}
			bufY := msg.Y + e.ScrollY
			bufX := textCol + e.ScrollX
			if bufY >= len(e.Lines) {
				bufY = len(e.Lines) - 1
			}
			if bufY < 0 {
				bufY = 0
			}
			lineLen := len([]rune(e.Lines[bufY]))
			if bufX > lineLen {
				bufX = lineLen
			}
			if bufX < 0 {
				bufX = 0
			}
			e.CursorY = bufY
			e.CursorX = bufX
			model.Buffers[model.BufIdx] = e
		} else if msg.X > dividerX {
			// Click in explorer panel — row 0 is "Files" header
			model.Focus = ExplorerPanel
			if msg.Y >= 1 {
				entryIdx := msg.Y - 1 + model.Explorer.ScrollY
				if entryIdx >= 0 && entryIdx < len(model.Explorer.Entries) {
					model.Explorer.Selected = entryIdx
					entry := model.Explorer.SelectedEntry()
					if entry != nil {
						if entry.IsDir {
							model.Explorer = model.Explorer.Toggle()
						} else {
							path := entry.Path
							name := entry.Name
							model.Status = fmt.Sprintf("Opening %s...", name)
							return model, func() Msg {
								data, err := os.ReadFile(path)
								if err != nil {
									return ErrorMsg{err}
								}
								return FileOpenedMsg{path, string(data)}
							}
						}
					}
				}
			}
			if explorerHeight > 0 {
				model.Explorer = model.Explorer.ScrollToView(explorerHeight)
			}
		}
	case 64: // scroll wheel up — 3 lines
		if model.Focus == EditorPanel {
			e := model.Buffers[model.BufIdx]
			for i := 0; i < 3; i++ {
				e = e.MoveUp()
			}
			e = e.ScrollToView(contentH, editorW-gutterWidth)
			model.Buffers[model.BufIdx] = e
		} else {
			for i := 0; i < 3; i++ {
				model.Explorer = model.Explorer.MoveUp()
			}
		}
	case 65: // scroll wheel down — 3 lines
		if model.Focus == EditorPanel {
			e := model.Buffers[model.BufIdx]
			for i := 0; i < 3; i++ {
				e = e.MoveDown()
			}
			e = e.ScrollToView(contentH, editorW-gutterWidth)
			model.Buffers[model.BufIdx] = e
		} else {
			for i := 0; i < 3; i++ {
				model.Explorer = model.Explorer.MoveDown()
			}
		}
	}
	return model, nil
}

func updateKey(model Model, key KeyEvent) (Model, Cmd) {
	// Search mode captures all input
	if model.SearchMode {
		return updateSearch(model, key)
	}

	// Handle C-x prefix sequence
	if model.Prefix == KeyCtrlX {
		model.Prefix = KeyNone
		switch key.Key {
		case KeyCtrlC:
			model.Quit = true
			return model, nil
		case KeyCtrlS:
			if model.Buffers[model.BufIdx].Filename == "" {
				model.Status = "No filename (open a file first)"
				return model, nil
			}
			model.Status = "Saving..."
			content := model.Buffers[model.BufIdx].ContentString()
			filename := model.Buffers[model.BufIdx].Filename
			return model, func() Msg {
				if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
					return ErrorMsg{err}
				}
				return FileSavedMsg{filename}
			}
		case KeyCtrlF:
			model.Focus = ExplorerPanel
			model.Status = "Select a file and press Enter"
			return model, nil
		case KeyRune:
			switch key.Char {
			case 'b':
				// Cycle to next buffer
				if len(model.Buffers) > 1 {
					model.BufIdx = (model.BufIdx + 1) % len(model.Buffers)
					model.Focus = EditorPanel
					name := model.Buffers[model.BufIdx].Filename
					if name == "" {
						name = "[scratch]"
					} else {
						name = filepath.Base(name)
					}
					model.Status = fmt.Sprintf("Buffer: %s [%d/%d]", name, model.BufIdx+1, len(model.Buffers))
				} else {
					model.Status = "No other buffers"
				}
				return model, nil
			case 'k':
				// Kill current buffer
				if len(model.Buffers) == 1 {
					// Replace with empty scratch rather than leaving 0 buffers
					model.Buffers[0] = NewEditorModel()
					model.Status = "Buffer cleared"
				} else {
					model.Buffers = append(model.Buffers[:model.BufIdx], model.Buffers[model.BufIdx+1:]...)
					if model.BufIdx >= len(model.Buffers) {
						model.BufIdx = len(model.Buffers) - 1
					}
					name := model.Buffers[model.BufIdx].Filename
					if name == "" {
						name = "[scratch]"
					} else {
						name = filepath.Base(name)
					}
					model.Status = fmt.Sprintf("Killed buffer. Now: %s [%d/%d]", name, model.BufIdx+1, len(model.Buffers))
				}
				return model, nil
			}
			model.Status = "C-x: unknown key"
			return model, nil
		default:
			model.Status = "C-x: unknown key"
			return model, nil
		}
	}

	// Start C-x prefix
	if key.Key == KeyCtrlX {
		model.Prefix = KeyCtrlX
		model.Status = "C-x ..."
		return model, nil
	}

	// Tab switches focus between panels
	if key.Key == KeyTab {
		if model.Focus == EditorPanel {
			model.Focus = ExplorerPanel
		} else {
			model.Focus = EditorPanel
		}
		return model, nil
	}

	// Dispatch to focused panel
	switch model.Focus {
	case EditorPanel:
		if key.Key == KeyCtrlS {
			return enterSearch(model), nil
		}
		return updateEditor(model, key), nil
	case ExplorerPanel:
		return updateExplorer(model, key)
	}
	return model, nil
}

func updateEditor(model Model, key KeyEvent) Model {
	e := model.Buffers[model.BufIdx]
	editorW, _, contentH := layoutSizes(model.Width, model.Height)
	textWidth := editorW - gutterWidth

	switch key.Key {
	case KeyCtrlSlash:
		e = e.Undo()
		model.Status = "Undo"
	case KeyRune:
		e = e.pushUndo()
		e = e.InsertRune(key.Char)
	case KeyEnter:
		e = e.pushUndo()
		e = e.InsertNewline()
	case KeyBackspace:
		e = e.pushUndo()
		e = e.DeleteBackward()
	case KeyDelete, KeyCtrlD:
		e = e.pushUndo()
		e = e.DeleteForward()
	case KeyCtrlK:
		e = e.pushUndo()
		e = e.KillLine()
	case KeyUp, KeyCtrlP:
		e = e.MoveUp()
	case KeyDown, KeyCtrlN:
		e = e.MoveDown()
	case KeyLeft, KeyCtrlB:
		e = e.MoveLeft()
	case KeyRight, KeyCtrlF:
		e = e.MoveRight()
	case KeyCtrlA, KeyHome:
		e = e.MoveToLineStart()
	case KeyCtrlE, KeyEnd:
		e = e.MoveToLineEnd()
	case KeyPgUp:
		for i := 0; i < contentH; i++ {
			e = e.MoveUp()
		}
	case KeyPgDown:
		for i := 0; i < contentH; i++ {
			e = e.MoveDown()
		}
	}

	e = e.ScrollToView(contentH, textWidth)
	model.Buffers[model.BufIdx] = e
	return model
}

// enterSearch activates incremental search mode
func enterSearch(model Model) Model {
	model.SearchMode = true
	model.SearchQuery = ""
	model.SearchOriginY = model.Buffers[model.BufIdx].CursorY
	model.SearchOriginX = model.Buffers[model.BufIdx].CursorX
	model.Status = "I-search: "
	return model
}

// updateSearch handles keypresses while incremental search is active
func updateSearch(model Model, key KeyEvent) (Model, Cmd) {
	switch key.Key {
	case KeyEscape, KeyCtrlG:
		// Cancel: restore cursor to where search began
		model.SearchMode = false
		e := model.Buffers[model.BufIdx]
		e.CursorY = model.SearchOriginY
		e.CursorX = model.SearchOriginX
		model.SearchQuery = ""
		editorW, _, contentH := layoutSizes(model.Width, model.Height)
		e = e.ScrollToView(contentH, editorW-gutterWidth)
		model.Buffers[model.BufIdx] = e
		model.Status = "Search cancelled"
		return model, nil
	case KeyEnter:
		// Confirm: keep cursor at matched position
		model.SearchMode = false
		if model.SearchQuery == "" {
			model.Status = ""
		} else {
			model.Status = fmt.Sprintf("Search: %s", model.SearchQuery)
		}
		model.SearchQuery = ""
		return model, nil
	case KeyCtrlS:
		// Advance to next match
		model = searchFrom(model, model.Buffers[model.BufIdx].CursorY, model.Buffers[model.BufIdx].CursorX+1)
		return model, nil
	case KeyBackspace:
		q := []rune(model.SearchQuery)
		if len(q) > 0 {
			model.SearchQuery = string(q[:len(q)-1])
		}
		model = searchFrom(model, model.SearchOriginY, model.SearchOriginX)
		return model, nil
	case KeyRune:
		model.SearchQuery += string(key.Char)
		model = searchFrom(model, model.SearchOriginY, model.SearchOriginX)
		return model, nil
	}
	return model, nil
}

// searchFrom scans forward from (startY, startX), wrapping around the buffer.
// It moves the editor cursor to the first match and updates the status line.
func searchFrom(model Model, startY, startX int) Model {
	q := []rune(model.SearchQuery)
	e := model.Buffers[model.BufIdx]
	if len(q) == 0 {
		e.CursorY = model.SearchOriginY
		e.CursorX = model.SearchOriginX
		model.Buffers[model.BufIdx] = e
		model.Status = "I-search: "
		return model
	}
	nLines := len(e.Lines)
	for dy := 0; dy < nLines; dy++ {
		lineIdx := (startY + dy) % nLines
		line := []rune(e.Lines[lineIdx])
		startCol := 0
		if dy == 0 {
			startCol = startX
			if startCol < 0 {
				startCol = 0
			}
		}
		for col := startCol; col+len(q) <= len(line); col++ {
			if runesMatch(line[col:col+len(q)], q) {
				e.CursorY = lineIdx
				e.CursorX = col
				editorW, _, contentH := layoutSizes(model.Width, model.Height)
				e = e.ScrollToView(contentH, editorW-gutterWidth)
				model.Buffers[model.BufIdx] = e
				model.Status = fmt.Sprintf("I-search: %s", model.SearchQuery)
				return model
			}
		}
	}
	model.Status = fmt.Sprintf("I-search: %s [not found]", model.SearchQuery)
	return model
}

func runesMatch(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func detectLineEnding(content string) LineEnding {
	if strings.Contains(content, "\r\n") {
		return LineEndingCRLF
	}
	return LineEndingLF
}

func normalizeLineEndings(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(content, "\r", "\n")
}

func updateExplorer(model Model, key KeyEvent) (Model, Cmd) {
	ex := model.Explorer
	explorerHeight := model.Height - 2 // -1 for status bar, -1 for header

	switch key.Key {
	case KeyUp, KeyCtrlP:
		ex = ex.MoveUp()
	case KeyDown, KeyCtrlN:
		ex = ex.MoveDown()
	case KeyEnter:
		entry := ex.SelectedEntry()
		if entry != nil {
			if entry.IsDir {
				ex = ex.Toggle()
			} else {
				path := entry.Path
				name := entry.Name
				model.Explorer = ex
				model.Status = fmt.Sprintf("Opening %s...", name)
				return model, func() Msg {
					data, err := os.ReadFile(path)
					if err != nil {
						return ErrorMsg{err}
					}
					return FileOpenedMsg{path, string(data)}
				}
			}
		}
	case KeyRight:
		entry := ex.SelectedEntry()
		if entry != nil && entry.IsDir && !entry.Open {
			ex = ex.Toggle()
		}
	case KeyLeft:
		entry := ex.SelectedEntry()
		if entry != nil && entry.IsDir && entry.Open {
			ex = ex.Toggle()
		}
	}

	ex = ex.ScrollToView(explorerHeight)
	model.Explorer = ex
	return model, nil
}
