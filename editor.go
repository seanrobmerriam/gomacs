package main

import "strings"

// EditorModel holds the state for the text editor panel
type EditorModel struct {
	Lines    []string
	CursorX  int // rune position in current line
	CursorY  int // line number
	ScrollY  int // first visible line
	ScrollX  int // first visible column
	Filename string
	Modified bool
	Lang     Language // syntax highlighting language
}

// NewEditorModel creates an empty editor buffer
func NewEditorModel() EditorModel {
	return EditorModel{
		Lines: []string{""},
	}
}

func (e EditorModel) currentLineRunes() []rune {
	return []rune(e.Lines[e.CursorY])
}

func (e EditorModel) clampCursorX() int {
	cx := e.CursorX
	if lineLen := len(e.currentLineRunes()); cx > lineLen {
		cx = lineLen
	}
	return cx
}

// InsertRune inserts a character at the cursor position
func (e EditorModel) InsertRune(ch rune) EditorModel {
	runes := e.currentLineRunes()
	cx := e.clampCursorX()

	newRunes := make([]rune, 0, len(runes)+1)
	newRunes = append(newRunes, runes[:cx]...)
	newRunes = append(newRunes, ch)
	newRunes = append(newRunes, runes[cx:]...)

	e.Lines[e.CursorY] = string(newRunes)
	e.CursorX = cx + 1
	e.Modified = true
	return e
}

// InsertNewline splits the current line at the cursor
func (e EditorModel) InsertNewline() EditorModel {
	runes := e.currentLineRunes()
	cx := e.clampCursorX()

	before := string(runes[:cx])
	after := string(runes[cx:])

	newLines := make([]string, 0, len(e.Lines)+1)
	newLines = append(newLines, e.Lines[:e.CursorY]...)
	newLines = append(newLines, before, after)
	if e.CursorY+1 < len(e.Lines) {
		newLines = append(newLines, e.Lines[e.CursorY+1:]...)
	}

	e.Lines = newLines
	e.CursorY++
	e.CursorX = 0
	e.Modified = true
	return e
}

// DeleteBackward deletes the character before the cursor, or joins lines
func (e EditorModel) DeleteBackward() EditorModel {
	if e.CursorX > 0 {
		runes := e.currentLineRunes()
		cx := e.clampCursorX()
		newRunes := make([]rune, 0, len(runes)-1)
		newRunes = append(newRunes, runes[:cx-1]...)
		newRunes = append(newRunes, runes[cx:]...)
		e.Lines[e.CursorY] = string(newRunes)
		e.CursorX = cx - 1
		e.Modified = true
	} else if e.CursorY > 0 {
		prevLine := e.Lines[e.CursorY-1]
		curLine := e.Lines[e.CursorY]
		newCursorX := len([]rune(prevLine))
		e.Lines[e.CursorY-1] = prevLine + curLine

		newLines := make([]string, 0, len(e.Lines)-1)
		newLines = append(newLines, e.Lines[:e.CursorY]...)
		newLines = append(newLines, e.Lines[e.CursorY+1:]...)
		e.Lines = newLines

		e.CursorY--
		e.CursorX = newCursorX
		e.Modified = true
	}
	return e
}

// DeleteForward deletes the character at the cursor, or joins with next line
func (e EditorModel) DeleteForward() EditorModel {
	runes := e.currentLineRunes()
	cx := e.clampCursorX()
	if cx < len(runes) {
		newRunes := make([]rune, 0, len(runes)-1)
		newRunes = append(newRunes, runes[:cx]...)
		newRunes = append(newRunes, runes[cx+1:]...)
		e.Lines[e.CursorY] = string(newRunes)
		e.Modified = true
	} else if e.CursorY < len(e.Lines)-1 {
		nextLine := e.Lines[e.CursorY+1]
		e.Lines[e.CursorY] = e.Lines[e.CursorY] + nextLine

		newLines := make([]string, 0, len(e.Lines)-1)
		newLines = append(newLines, e.Lines[:e.CursorY+1]...)
		newLines = append(newLines, e.Lines[e.CursorY+2:]...)
		e.Lines = newLines
		e.Modified = true
	}
	return e
}

// KillLine kills text from cursor to end of line, or joins with next line
func (e EditorModel) KillLine() EditorModel {
	runes := e.currentLineRunes()
	cx := e.clampCursorX()
	if cx < len(runes) {
		e.Lines[e.CursorY] = string(runes[:cx])
		e.Modified = true
	} else if e.CursorY < len(e.Lines)-1 {
		nextLine := e.Lines[e.CursorY+1]
		e.Lines[e.CursorY] = e.Lines[e.CursorY] + nextLine

		newLines := make([]string, 0, len(e.Lines)-1)
		newLines = append(newLines, e.Lines[:e.CursorY+1]...)
		newLines = append(newLines, e.Lines[e.CursorY+2:]...)
		e.Lines = newLines
		e.Modified = true
	}
	return e
}

// MoveUp moves the cursor one line up
func (e EditorModel) MoveUp() EditorModel {
	if e.CursorY > 0 {
		e.CursorY--
		if lineLen := len([]rune(e.Lines[e.CursorY])); e.CursorX > lineLen {
			e.CursorX = lineLen
		}
	}
	return e
}

// MoveDown moves the cursor one line down
func (e EditorModel) MoveDown() EditorModel {
	if e.CursorY < len(e.Lines)-1 {
		e.CursorY++
		if lineLen := len([]rune(e.Lines[e.CursorY])); e.CursorX > lineLen {
			e.CursorX = lineLen
		}
	}
	return e
}

// MoveLeft moves the cursor one character left, wrapping to previous line
func (e EditorModel) MoveLeft() EditorModel {
	if e.CursorX > 0 {
		e.CursorX--
	} else if e.CursorY > 0 {
		e.CursorY--
		e.CursorX = len([]rune(e.Lines[e.CursorY]))
	}
	return e
}

// MoveRight moves the cursor one character right, wrapping to next line
func (e EditorModel) MoveRight() EditorModel {
	lineLen := len([]rune(e.Lines[e.CursorY]))
	if e.CursorX < lineLen {
		e.CursorX++
	} else if e.CursorY < len(e.Lines)-1 {
		e.CursorY++
		e.CursorX = 0
	}
	return e
}

// MoveToLineStart moves the cursor to the beginning of the line
func (e EditorModel) MoveToLineStart() EditorModel {
	e.CursorX = 0
	return e
}

// MoveToLineEnd moves the cursor to the end of the line
func (e EditorModel) MoveToLineEnd() EditorModel {
	e.CursorX = len([]rune(e.Lines[e.CursorY]))
	return e
}

// ScrollToView adjusts scroll offsets so the cursor is visible
func (e EditorModel) ScrollToView(height, textWidth int) EditorModel {
	if e.CursorY < e.ScrollY {
		e.ScrollY = e.CursorY
	}
	if e.CursorY >= e.ScrollY+height {
		e.ScrollY = e.CursorY - height + 1
	}
	if textWidth > 0 {
		if e.CursorX < e.ScrollX {
			e.ScrollX = e.CursorX
		}
		if e.CursorX >= e.ScrollX+textWidth {
			e.ScrollX = e.CursorX - textWidth + 1
		}
	}
	return e
}

// ContentString returns the buffer contents as a single string
func (e EditorModel) ContentString() string {
	return strings.Join(e.Lines, "\n")
}
