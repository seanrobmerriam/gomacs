package main

import (
	"bytes"
	"fmt"
	"os"
)

// Color represents a terminal color
type Color int

const (
	ColorDefault Color = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

// Style represents text styling attributes
type Style struct {
	FG      Color
	BG      Color
	Bold    bool
	Reverse bool
}

// DefaultStyle is unstyled text
var DefaultStyle = Style{}

// Cell represents a single character on screen with styling
type Cell struct {
	Ch    rune
	Style Style
}

// Screen is a 2D grid of cells representing the terminal display
type Screen struct {
	Cells         [][]Cell
	Width, Height int
}

// NewScreen creates a blank screen filled with spaces
func NewScreen(width, height int) *Screen {
	cells := make([][]Cell, height)
	for y := range cells {
		cells[y] = make([]Cell, width)
		for x := range cells[y] {
			cells[y][x] = Cell{Ch: ' ', Style: DefaultStyle}
		}
	}
	return &Screen{Cells: cells, Width: width, Height: height}
}

// Set places a character with style at (x, y), bounds-checked
func (s *Screen) Set(x, y int, ch rune, style Style) {
	if x >= 0 && x < s.Width && y >= 0 && y < s.Height {
		s.Cells[y][x] = Cell{Ch: ch, Style: style}
	}
}

// SetString writes a string at (x, y), clipping at the right edge
func (s *Screen) SetString(x, y int, str string, style Style) {
	for _, ch := range str {
		if x >= s.Width {
			break
		}
		s.Set(x, y, ch, style)
		x++
	}
}

// Render writes the screen to out using ANSI escape sequences.
// If prev is non-nil and same dimensions, only changed cells are written.
func (s *Screen) Render(out *os.File, prev *Screen) {
	var buf bytes.Buffer

	// Hide cursor during render
	buf.WriteString("\x1b[?25l")

	if prev == nil || prev.Width != s.Width || prev.Height != s.Height {
		// Full render — write every cell, line by line
		for y := 0; y < s.Height; y++ {
			fmt.Fprintf(&buf, "\x1b[%d;1H", y+1)
			lastStyle := Style{FG: -1} // impossible style forces first write
			for x := 0; x < s.Width; x++ {
				cell := s.Cells[y][x]
				if cell.Style != lastStyle {
					buf.WriteString("\x1b[0m")
					writeStyle(&buf, cell.Style)
					lastStyle = cell.Style
				}
				buf.WriteRune(cell.Ch)
			}
			buf.WriteString("\x1b[0m")
		}
	} else {
		// Diff render — only write changed cells
		for y := 0; y < s.Height; y++ {
			for x := 0; x < s.Width; x++ {
				cell := s.Cells[y][x]
				if prev.Cells[y][x] == cell {
					continue
				}
				fmt.Fprintf(&buf, "\x1b[%d;%dH\x1b[0m", y+1, x+1)
				writeStyle(&buf, cell.Style)
				buf.WriteRune(cell.Ch)
			}
		}
		buf.WriteString("\x1b[0m")
	}

	out.Write(buf.Bytes())
}

func writeStyle(buf *bytes.Buffer, s Style) {
	if s.Bold {
		buf.WriteString("\x1b[1m")
	}
	if s.Reverse {
		buf.WriteString("\x1b[7m")
	}
	if s.FG > ColorDefault {
		fmt.Fprintf(buf, "\x1b[%dm", 29+int(s.FG))
	}
	if s.BG > ColorDefault {
		fmt.Fprintf(buf, "\x1b[%dm", 39+int(s.BG))
	}
}

// SetCursor moves the terminal cursor to (x,y) and shows it
func SetCursor(out *os.File, x, y int) {
	fmt.Fprintf(out, "\x1b[%d;%dH\x1b[?25h", y+1, x+1)
}

// ClearScreen clears the terminal and moves cursor to top-left
func ClearScreen(out *os.File) {
	fmt.Fprint(out, "\x1b[2J\x1b[H")
}
