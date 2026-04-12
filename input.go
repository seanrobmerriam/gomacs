package main

import (
	"os"
	"unicode/utf8"
)

// Key represents a keyboard key type
type Key int

const (
	KeyNone Key = iota
	KeyRune
	KeyEnter
	KeyBackspace
	KeyTab
	KeyEscape
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDown
	KeyDelete
	KeyCtrlA
	KeyCtrlB
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyCtrlF
	KeyCtrlG
	KeyCtrlK
	KeyCtrlL
	KeyCtrlN
	KeyCtrlP
	KeyCtrlS
	KeyCtrlV
	KeyCtrlX
)

// KeyEvent represents a single key press
type KeyEvent struct {
	Key  Key
	Char rune
}

// ReadKey reads a single key event from the given file (stdin)
func ReadKey(r *os.File) KeyEvent {
	var buf [16]byte
	n, err := r.Read(buf[:])
	if err != nil || n == 0 {
		return KeyEvent{Key: KeyNone}
	}
	b := buf[:n]

	if n == 1 {
		return parseSingleByte(b[0])
	}

	// Escape sequences
	if b[0] == 0x1b {
		return parseEscape(b)
	}

	// UTF-8 multi-byte character
	r2, _ := utf8.DecodeRune(b)
	if r2 != utf8.RuneError {
		return KeyEvent{Key: KeyRune, Char: r2}
	}

	return KeyEvent{Key: KeyNone}
}

func parseSingleByte(ch byte) KeyEvent {
	switch {
	case ch == 0x01:
		return KeyEvent{Key: KeyCtrlA}
	case ch == 0x02:
		return KeyEvent{Key: KeyCtrlB}
	case ch == 0x03:
		return KeyEvent{Key: KeyCtrlC}
	case ch == 0x04:
		return KeyEvent{Key: KeyCtrlD}
	case ch == 0x05:
		return KeyEvent{Key: KeyCtrlE}
	case ch == 0x06:
		return KeyEvent{Key: KeyCtrlF}
	case ch == 0x07:
		return KeyEvent{Key: KeyCtrlG}
	case ch == 0x0b:
		return KeyEvent{Key: KeyCtrlK}
	case ch == 0x0c:
		return KeyEvent{Key: KeyCtrlL}
	case ch == 0x0e:
		return KeyEvent{Key: KeyCtrlN}
	case ch == 0x10:
		return KeyEvent{Key: KeyCtrlP}
	case ch == 0x13:
		return KeyEvent{Key: KeyCtrlS}
	case ch == 0x16:
		return KeyEvent{Key: KeyCtrlV}
	case ch == 0x18:
		return KeyEvent{Key: KeyCtrlX}
	case ch == 0x0d || ch == 0x0a:
		return KeyEvent{Key: KeyEnter}
	case ch == 0x09:
		return KeyEvent{Key: KeyTab}
	case ch == 0x7f || ch == 0x08:
		return KeyEvent{Key: KeyBackspace}
	case ch == 0x1b:
		return KeyEvent{Key: KeyEscape}
	case ch >= 0x20 && ch < 0x7f:
		return KeyEvent{Key: KeyRune, Char: rune(ch)}
	default:
		return KeyEvent{Key: KeyNone}
	}
}

func parseEscape(b []byte) KeyEvent {
	if len(b) < 3 || b[1] != '[' {
		return KeyEvent{Key: KeyEscape}
	}
	switch b[2] {
	case 'A':
		return KeyEvent{Key: KeyUp}
	case 'B':
		return KeyEvent{Key: KeyDown}
	case 'C':
		return KeyEvent{Key: KeyRight}
	case 'D':
		return KeyEvent{Key: KeyLeft}
	case 'H':
		return KeyEvent{Key: KeyHome}
	case 'F':
		return KeyEvent{Key: KeyEnd}
	case '3':
		if len(b) >= 4 && b[3] == '~' {
			return KeyEvent{Key: KeyDelete}
		}
	case '5':
		if len(b) >= 4 && b[3] == '~' {
			return KeyEvent{Key: KeyPgUp}
		}
	case '6':
		if len(b) >= 4 && b[3] == '~' {
			return KeyEvent{Key: KeyPgDown}
		}
	}
	return KeyEvent{Key: KeyEscape}
}
