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

// --- Model ---

// Model is the complete application state
type Model struct {
	Editor        EditorModel
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
		Editor:   NewEditorModel(),
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
		model.Editor.Modified = false
		model.Status = fmt.Sprintf("Saved %s", filepath.Base(msg.Filename))
		return model, nil

	case FileOpenedMsg:
		lines := strings.Split(msg.Content, "\n")
		if len(lines) == 0 {
			lines = []string{""}
		}
		model.Editor = EditorModel{
			Lines:    lines,
			Filename: msg.Filename,
			Lang:     DetectLanguage(msg.Filename),
		}
		model.Focus = EditorPanel
		model.Status = fmt.Sprintf("Opened %s", filepath.Base(msg.Filename))
		return model, nil

	case ErrorMsg:
		model.Status = fmt.Sprintf("Error: %v", msg.Err)
		return model, nil
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
			if model.Editor.Filename == "" {
				model.Status = "No filename (open a file first)"
				return model, nil
			}
			model.Status = "Saving..."
			content := model.Editor.ContentString()
			filename := model.Editor.Filename
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
	e := model.Editor
	editorW, _, contentH := layoutSizes(model.Width, model.Height)
	textWidth := editorW - gutterWidth

	switch key.Key {
	case KeyRune:
		e = e.InsertRune(key.Char)
	case KeyEnter:
		e = e.InsertNewline()
	case KeyBackspace:
		e = e.DeleteBackward()
	case KeyDelete, KeyCtrlD:
		e = e.DeleteForward()
	case KeyCtrlK:
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
	model.Editor = e
	return model
}

// enterSearch activates incremental search mode
func enterSearch(model Model) Model {
	model.SearchMode = true
	model.SearchQuery = ""
	model.SearchOriginY = model.Editor.CursorY
	model.SearchOriginX = model.Editor.CursorX
	model.Status = "I-search: "
	return model
}

// updateSearch handles keypresses while incremental search is active
func updateSearch(model Model, key KeyEvent) (Model, Cmd) {
	switch key.Key {
	case KeyEscape, KeyCtrlG:
		// Cancel: restore cursor to where search began
		model.SearchMode = false
		model.Editor.CursorY = model.SearchOriginY
		model.Editor.CursorX = model.SearchOriginX
		model.SearchQuery = ""
		_, _, contentH := layoutSizes(model.Width, model.Height)
		editorW, _, _ := layoutSizes(model.Width, model.Height)
		model.Editor = model.Editor.ScrollToView(contentH, editorW-gutterWidth)
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
		model = searchFrom(model, model.Editor.CursorY, model.Editor.CursorX+1)
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
	if len(q) == 0 {
		model.Editor.CursorY = model.SearchOriginY
		model.Editor.CursorX = model.SearchOriginX
		model.Status = "I-search: "
		return model
	}
	nLines := len(model.Editor.Lines)
	for dy := 0; dy < nLines; dy++ {
		lineIdx := (startY + dy) % nLines
		line := []rune(model.Editor.Lines[lineIdx])
		startCol := 0
		if dy == 0 {
			startCol = startX
			if startCol < 0 {
				startCol = 0
			}
		}
		for col := startCol; col+len(q) <= len(line); col++ {
			if runesMatch(line[col:col+len(q)], q) {
				model.Editor.CursorY = lineIdx
				model.Editor.CursorX = col
				editorW, _, contentH := layoutSizes(model.Width, model.Height)
				model.Editor = model.Editor.ScrollToView(contentH, editorW-gutterWidth)
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
