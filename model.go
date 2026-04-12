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
	Editor   EditorModel
	Explorer ExplorerModel
	Focus    Panel
	Width    int
	Height   int
	Quit     bool
	Status   string
	Prefix   Key // for multi-key sequences (e.g., C-x)
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
