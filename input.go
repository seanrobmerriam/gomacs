package main

import (
	"io"
	"os"
	"syscall"
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
	KeyCtrlTab
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
	KeyCtrlSlash // C-/ or C-_
)

// KeyEvent represents a single key press
type KeyEvent struct {
	Key  Key
	Char rune
}

// ReadInput reads a single input event from stdin.
// Returns a KeyMsg, MouseMsg, or nil if no meaningful event occurred.
func ReadInput(r *os.File) Msg {
	var buf [16]byte
	n, err := r.Read(buf[:1])
	if err != nil || n == 0 {
		return nil
	}

	// Escape sequences can arrive split across multiple reads. Drain any bytes
	// already queued so mouse reports and CSI sequences do not leak into the
	// buffer as literal characters.
	if buf[0] == 0x1b {
		n += readPendingBytes(r, buf[n:])

		// X10 mouse protocol is always 6 bytes total. If we have recognized the
		// prefix but not the full payload yet, block for the remaining bytes.
		if n >= 3 && buf[1] == '[' && buf[2] == 'M' && n < 6 {
			m, err := io.ReadFull(r, buf[n:6])
			n += m
			if err != nil && n < 6 {
				return nil
			}
		}
	}

	// UTF-8 runes may also span multiple bytes.
	if buf[0] >= 0x80 && buf[0] != 0x1b {
		n += readPendingBytes(r, buf[n:])
	}

	return parseInputBytes(buf[:n])
}

func readPendingBytes(r *os.File, dst []byte) int {
	if len(dst) == 0 {
		return 0
	}

	fd := int(r.Fd())
	if err := syscall.SetNonblock(fd, true); err != nil {
		return 0
	}
	defer syscall.SetNonblock(fd, false)

	total := 0
	for total < len(dst) {
		n, err := r.Read(dst[total:])
		if n > 0 {
			total += n
			continue
		}
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			break
		}
		break
	}
	return total
}

func parseInputBytes(b []byte) Msg {
	if len(b) == 0 {
		return nil
	}

	if len(b) == 1 {
		ke := parseSingleByte(b[0])
		if ke.Key == KeyNone {
			return nil
		}
		return KeyMsg{ke}
	}

	// Escape sequences
	if b[0] == 0x1b {
		// X10 mouse protocol: ESC [ M <btn+32> <col+33> <row+33>
		if len(b) >= 6 && b[1] == '[' && b[2] == 'M' {
			btn := int(b[3]) - 32
			x := int(b[4]) - 33
			y := int(b[5]) - 33
			if x < 0 {
				x = 0
			}
			if y < 0 {
				y = 0
			}
			return MouseMsg{Btn: btn, X: x, Y: y}
		}
		ke := parseEscape(b)
		if ke.Key == KeyNone {
			return nil
		}
		return KeyMsg{ke}
	}

	// UTF-8 multi-byte character
	r2, _ := utf8.DecodeRune(b)
	if r2 != utf8.RuneError {
		return KeyMsg{KeyEvent{Key: KeyRune, Char: r2}}
	}

	return nil
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
	case ch == 0x1f:
		return KeyEvent{Key: KeyCtrlSlash}
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
	// Ctrl+Tab is not standardized across terminals; support common CSI variants.
	if string(b) == "\x1b[1;5I" || string(b) == "\x1b[27;5;9~" || string(b) == "\x1b[9;5u" {
		return KeyEvent{Key: KeyCtrlTab}
	}

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
