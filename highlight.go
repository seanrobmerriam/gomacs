package main

// Language identifies the syntax highlighting language for a buffer.
type Language int

const (
	LangPlain  Language = iota // no highlighting
	LangGo                     // .go
	LangC                      // .c .h .cpp .cc .cxx .hpp .hxx
	LangPython                 // .py .pyw
	LangJS                     // .js .mjs .ts .tsx .jsx
	LangShell                  // .sh .bash .zsh .fish
)

// token style palette — read-only after init
var (
	styleKeyword = Style{FG: ColorBlue, Bold: true}
	styleBuiltin = Style{FG: ColorMagenta}
	styleString  = Style{FG: ColorGreen}
	styleComment = Style{FG: ColorCyan}
	styleNumber  = Style{FG: ColorYellow}
)

// --- Keyword sets (read-only after init) ---

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true,
	"continue": true, "default": true, "defer": true, "else": true,
	"fallthrough": true, "for": true, "func": true, "go": true,
	"goto": true, "if": true, "import": true, "interface": true,
	"map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true,
	"var": true,
}

var goBuiltins = map[string]bool{
	"append": true, "cap": true, "close": true, "complex": true,
	"copy": true, "delete": true, "error": true, "false": true,
	"imag": true, "iota": true, "len": true, "make": true,
	"new": true, "nil": true, "panic": true, "print": true,
	"println": true, "real": true, "recover": true, "true": true,
	"any": true, "bool": true, "byte": true, "comparable": true,
	"float32": true, "float64": true, "int": true, "int8": true,
	"int16": true, "int32": true, "int64": true, "rune": true,
	"string": true, "uint": true, "uint8": true, "uint16": true,
	"uint32": true, "uint64": true, "uintptr": true,
}

var cKeywords = map[string]bool{
	"auto": true, "break": true, "case": true, "char": true,
	"const": true, "continue": true, "default": true, "do": true,
	"double": true, "else": true, "enum": true, "extern": true,
	"float": true, "for": true, "goto": true, "if": true,
	"inline": true, "int": true, "long": true, "register": true,
	"return": true, "short": true, "signed": true, "sizeof": true,
	"static": true, "struct": true, "switch": true, "typedef": true,
	"union": true, "unsigned": true, "void": true, "volatile": true,
	"while": true,
	// C++ additions
	"bool": true, "catch": true, "class": true, "delete": true,
	"explicit": true, "false": true, "friend": true, "mutable": true,
	"namespace": true, "new": true, "nullptr": true, "operator": true,
	"private": true, "protected": true, "public": true, "template": true,
	"this": true, "throw": true, "true": true, "try": true,
	"typename": true, "using": true, "virtual": true,
}

var pyKeywords = map[string]bool{
	"False": true, "None": true, "True": true, "and": true,
	"as": true, "assert": true, "async": true, "await": true,
	"break": true, "class": true, "continue": true, "def": true,
	"del": true, "elif": true, "else": true, "except": true,
	"finally": true, "for": true, "from": true, "global": true,
	"if": true, "import": true, "in": true, "is": true,
	"lambda": true, "nonlocal": true, "not": true, "or": true,
	"pass": true, "raise": true, "return": true, "try": true,
	"while": true, "with": true, "yield": true,
}

var jsKeywords = map[string]bool{
	"break": true, "case": true, "catch": true, "class": true,
	"const": true, "continue": true, "debugger": true, "default": true,
	"delete": true, "do": true, "else": true, "export": true,
	"extends": true, "finally": true, "for": true, "function": true,
	"if": true, "import": true, "in": true, "instanceof": true,
	"let": true, "new": true, "of": true, "return": true,
	"static": true, "super": true, "switch": true, "this": true,
	"throw": true, "try": true, "typeof": true, "var": true,
	"void": true, "while": true, "with": true, "yield": true,
	"async": true, "await": true, "from": true, "as": true,
	"true": true, "false": true, "null": true, "undefined": true,
}

var shellKeywords = map[string]bool{
	"if": true, "then": true, "else": true, "elif": true, "fi": true,
	"for": true, "while": true, "do": true, "done": true, "case": true,
	"esac": true, "function": true, "in": true, "select": true,
	"until": true, "return": true, "break": true, "continue": true,
	"local": true, "export": true, "readonly": true, "unset": true,
	"shift": true, "source": true, "echo": true, "exit": true,
}

// DetectLanguage returns a Language from the file extension of filename.
// No standard library imports — manual extension scan.
func DetectLanguage(filename string) Language {
	// Find the rightmost '.' after the last '/'
	dot := -1
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '/' {
			break
		}
		if filename[i] == '.' {
			dot = i
			break
		}
	}
	if dot < 0 || dot == len(filename)-1 {
		return LangPlain
	}
	switch filename[dot+1:] {
	case "go":
		return LangGo
	case "c", "h", "cpp", "cc", "cxx", "hpp", "hxx":
		return LangC
	case "py", "pyw":
		return LangPython
	case "js", "mjs", "cjs", "ts", "tsx", "jsx":
		return LangJS
	case "sh", "bash", "zsh", "fish":
		return LangShell
	}
	return LangPlain
}

// Highlight tokenizes a single source line and returns one Style per rune.
// No state is preserved across lines (MVP line-by-line tokenization).
func Highlight(lang Language, line string) []Style {
	runes := []rune(line)
	out := make([]Style, len(runes))
	if lang == LangPlain || len(runes) == 0 {
		return out
	}
	highlightLine(lang, runes, out)
	return out
}

func highlightLine(lang Language, runes []rune, out []Style) {
	n := len(runes)
	i := 0
	for i < n {
		ch := runes[i]

		// --- Line comment ---
		if isLineCommentStart(lang, runes, i) {
			for ; i < n; i++ {
				out[i] = styleComment
			}
			return
		}

		// --- Block comment (Go, C, JS): /* ... */ ---
		if isBlockCommentStart(lang, runes, i) {
			out[i] = styleComment
			i++
			out[i] = styleComment
			i++
			for i < n {
				if i+1 < n && runes[i] == '*' && runes[i+1] == '/' {
					out[i] = styleComment
					out[i+1] = styleComment
					i += 2
					break
				}
				out[i] = styleComment
				i++
			}
			continue
		}

		// --- Go raw string (backtick) ---
		if lang == LangGo && ch == '`' {
			out[i] = styleString
			i++
			for i < n {
				out[i] = styleString
				if runes[i] == '`' {
					i++
					break
				}
				i++
			}
			continue
		}

		// --- Quoted string or character literal ---
		if ch == '"' || ch == '\'' {
			quote := ch
			out[i] = styleString
			i++
			for i < n {
				// Escape sequence (not in shell)
				if runes[i] == '\\' && lang != LangShell {
					out[i] = styleString
					i++
					if i < n {
						out[i] = styleString
						i++
					}
					continue
				}
				out[i] = styleString
				if runes[i] == quote {
					i++
					break
				}
				i++
			}
			continue
		}

		// --- Numeric literal ---
		if isDigit(ch) || (ch == '.' && i+1 < n && isDigit(runes[i+1])) {
			end := scanNumber(runes, i)
			for ; i < end; i++ {
				out[i] = styleNumber
			}
			continue
		}

		// --- Identifier, keyword, or builtin ---
		if isIdentStart(ch) {
			start := i
			for i < n && isIdentChar(runes[i]) {
				i++
			}
			word := string(runes[start:i])
			st := wordStyle(lang, word)
			for j := start; j < i; j++ {
				out[j] = st
			}
			continue
		}

		i++
	}
}

func isLineCommentStart(lang Language, runes []rune, i int) bool {
	ch := runes[i]
	n := len(runes)
	switch lang {
	case LangGo, LangJS:
		return i+1 < n && ch == '/' && runes[i+1] == '/'
	case LangC:
		// C: both // comments and # preprocessor directives
		return ch == '#' || (i+1 < n && ch == '/' && runes[i+1] == '/')
	case LangPython, LangShell:
		return ch == '#'
	}
	return false
}

func isBlockCommentStart(lang Language, runes []rune, i int) bool {
	switch lang {
	case LangGo, LangC, LangJS:
		return i+1 < len(runes) && runes[i] == '/' && runes[i+1] == '*'
	}
	return false
}

// wordStyle returns the highlight style for an identifier word.
// Returns DefaultStyle for plain identifiers.
func wordStyle(lang Language, word string) Style {
	switch lang {
	case LangGo:
		if goKeywords[word] {
			return styleKeyword
		}
		if goBuiltins[word] {
			return styleBuiltin
		}
	case LangC:
		if cKeywords[word] {
			return styleKeyword
		}
	case LangPython:
		if pyKeywords[word] {
			return styleKeyword
		}
	case LangJS:
		if jsKeywords[word] {
			return styleKeyword
		}
	case LangShell:
		if shellKeywords[word] {
			return styleKeyword
		}
	}
	return DefaultStyle
}

// scanNumber returns the index just past the end of the numeric literal
// starting at runes[i]. Handles decimal, float, 0x hex, 0b binary.
func scanNumber(runes []rune, i int) int {
	n := len(runes)
	// 0x / 0b prefix
	if runes[i] == '0' && i+1 < n {
		switch runes[i+1] {
		case 'x', 'X':
			i += 2
			for i < n && isHexDigit(runes[i]) {
				i++
			}
			return i
		case 'b', 'B':
			i += 2
			for i < n && (runes[i] == '0' || runes[i] == '1' || runes[i] == '_') {
				i++
			}
			return i
		}
	}
	// Decimal integer part
	for i < n && (isDigit(runes[i]) || runes[i] == '_') {
		i++
	}
	// Fractional part
	if i < n && runes[i] == '.' && i+1 < n && isDigit(runes[i+1]) {
		i++
		for i < n && (isDigit(runes[i]) || runes[i] == '_') {
			i++
		}
	}
	// Exponent
	if i < n && (runes[i] == 'e' || runes[i] == 'E') {
		i++
		if i < n && (runes[i] == '+' || runes[i] == '-') {
			i++
		}
		for i < n && isDigit(runes[i]) {
			i++
		}
	}
	return i
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch rune) bool {
	return isDigit(ch) ||
		(ch >= 'a' && ch <= 'f') ||
		(ch >= 'A' && ch <= 'F') ||
		ch == '_'
}

func isIdentStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentChar(ch rune) bool {
	return isIdentStart(ch) || isDigit(ch)
}
