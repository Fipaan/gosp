package log

import (
	"fmt"
	"runtime"
	"io"
	"os"
	"unicode"
	"strconv"
	"path/filepath"
)
const DEBUG = true
const COLORED = true
const PRINT_CHAR_BASE = 10

type ControlKey uint16
const (
	NUL ControlKey = 0 
	SOH = 1 
	STX = 2 
	ETX = 3 
	EOT = 4 
	ENQ = 5 
	ACK = 6 
	BEL = 7 
	BS  = 8 
	HT  = 9 
	LF  = 10
	VT  = 11
	FF  = 12
	CR  = 13
	SO  = 14
	SI  = 15
	DLE = 16
	DC1 = 17
	DC2 = 18
	DC3 = 19
	DC4 = 20
	NAK = 21
	SYN = 22
	ETB = 23
	CAN = 24
	EM  = 25
	SUB = 26
	ESC = 27
	FS  = 28
	GS  = 29
	RS  = 30
	US  = 31
	DEL = 127
)
const ESC_STR = string(ESC)

type Color uint16
const (
	ColorDefault Color = iota
	Black
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
	BrightBlack
	BrightRed
	BrightGreen
	BrightYellow
	BrightBlue
	BrightMagenta
	BrightCyan
	BrightWhite
)
func (c Color) foreCode() string {
	switch c {
	case ColorDefault:  return "39"
	case Black:         return "30"
	case Red:           return "31"
	case Green:         return "32"
	case Yellow:        return "33"
	case Blue:          return "34"
	case Magenta:       return "35"
	case Cyan:          return "36"
	case White:         return "37"
	case BrightBlack:   return "90"
	case BrightRed:     return "91"
	case BrightGreen:   return "92"
	case BrightYellow:  return "93"
	case BrightBlue:    return "94"
	case BrightMagenta: return "95"
	case BrightCyan:    return "96"
	case BrightWhite:   return "97"
	default: Unreachable("Unknown color: %d", uint16(c))
	}
	return ""
}
func (c Color) foreEscaped() string {
	return ESC_STR + "[" + c.foreCode() + "m"
}
func (c Color) Colorf(format string, args ...any) string {
	if COLORED {
		return fmt.Sprintf(c.foreEscaped() + format + ColorDefault.foreEscaped(), args...)
	}
	return fmt.Sprintf(format, args...)
}

type charDesc struct {
	compact, full, desc string
}
func CharDesc(ch rune, compact bool) (string, string) {
	var chVal string
	if ch < 32 {
		cDesc := charDescs[ch]
		if compact { chVal = cDesc.compact } else { chVal = cDesc.full }
		return chVal, cDesc.desc
	} else if ch == DEL {
		if compact { chVal = "DEL" } else { chVal = "<DEL>" }
		return chVal, "Delete"
	}
	if !unicode.IsPrint(ch) {
		chVal := strconv.FormatUint(uint64(ch), PRINT_CHAR_BASE)
		if !compact { chVal = "<" + chVal + ">" }
		return chVal, "non-printable"
	}
	return string(ch), "printable"
}
func Str2Printable(str string) (result string) {
	runes	:= []rune(str)
	runesLen := len(runes)
	for i := 0; i < runesLen; i++ {
		ch, _ := CharDesc(runes[i], false)
		result += ch
	}
	return result
}

func Fprintf(w io.Writer, skip int, format string, args ...any) {
	if skip >= 0 {
		_, file, line, ok := runtime.Caller(skip + 1)
		if !ok {
			file = "?"
			line = 0
		}
		if wd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(wd, file); err == nil {
				file = "./" + rel
			}
		}
		fmt.Fprintf(w, "%s:%d: ", file, line)
	}
	fmt.Fprintf(w, format, args...)
}
func Printf(format string, args ...any) {
	Fprintf(os.Stdout, -1, format, args...)
}
func Escapedf(format string, args ...any) {
	Printf(ESC_STR + format, args...)
}
func Errorf(format string, args ...any) {
	Fprintf(os.Stderr, -1, "ERROR: " + format + "\r\n", args...)
}
func Infof(format string, args ...any) {
	Fprintf(os.Stdout, -1, "INFO: " + format + "\r\n", args...)
}
func Debugf(format string, args ...any) {
	if DEBUG {
		Fprintf(os.Stdout, -1, "DEBUG: " + format + "\r\n", args...)
	}
}
func Abortf(format string, args ...any) {
	abortf(1, format, args...)
}
func Todof(format string, args ...any) {
	Fprintf(os.Stderr, 1, "TODO: " + format + "\r\n", args...)
	os.Exit(1)
}
func Unreachable(format string, args ...any) {
	abortf(1, "unreachable: " + format, args...)
}
func Assert(cond bool, msg string) {
	if !cond { abortf(1, "assertion failed: %s", msg) }
}

func abortf(skip int, format string, args ...any) {
	Fprintf(os.Stderr, skip + 1, "ABORT: " + format + "\r\n", args...)
	os.Exit(1)
}

var charDescs = []charDesc {
	charDesc{"\\0", "\\0",   "null character"},
	charDesc{"SOH", "<SOH>", "start of heading"},
	charDesc{"STX", "<STX>", "start of text"},
	charDesc{"ETX", "<ETX>", "end of text"},
	charDesc{"EOT", "<EOT>", "end of transmission"},
	charDesc{"ENQ", "<ENQ>", "enquiry"},
	charDesc{"ACK", "<ACK>", "acknowledge"},
	charDesc{"BEL", "<BEL>", "bell"},
	charDesc{"\\b", "\\b",   "backspace"},
	charDesc{"\\t", "\\t",   "horizontal tab"},
	charDesc{"\\n", "\\n",   "new line"},
	charDesc{"\\v", "\\v",   "vertical tab"},
	charDesc{"\\f", "\\f",   "form feed"},
	charDesc{"\\r", "\\r",   "carriage ret"},
	charDesc{"SO",  "<SO>",  "shift out"},
	charDesc{"SI",  "<SI>",  "shift in"},
	charDesc{"DLE", "<DLE>", "data link escape"},
	charDesc{"DC1", "<DC1>", "device control 1"},
	charDesc{"DC2", "<DC2>", "device control 2"},
	charDesc{"DC3", "<DC3>", "device control 3"},
	charDesc{"DC4", "<DC4>", "device control 4"},
	charDesc{"NAK", "<NAK>", "negative ack."},
	charDesc{"SYN", "<SYN>", "synchronous idle"},
	charDesc{"ETB", "<ETB>", "end of trans. blk"},
	charDesc{"CAN", "<CAN>", "cancel"},
	charDesc{"EM",  "<EM>",  "end of medium"},
	charDesc{"SUB", "<SUB>", "substitute"},
	charDesc{"ESC", "<ESC>", "escape"},
	charDesc{"FS",  "<FS>",  "file separator"},
	charDesc{"GS",  "<GS>",  "group separator"},
	charDesc{"RS",  "<RS>",  "record separator"},
	charDesc{"US",  "<US>",  "unit separator"},
}
