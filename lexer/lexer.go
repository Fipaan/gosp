package lexer

import (
    "github.com/Fipaan/gosp/log"
    "os"
    "fmt"
    "strconv"
    "unicode"
)

func IsAlphaNum(ch rune) bool {
    return unicode.IsLetter(ch) ||
           unicode.IsDigit(ch)  || ch == '_'
}

var ID_CHARS_SPECIAL = []rune("+-/*.:_=!<>|&")
func IsIdFirst(ch rune) bool {
    if unicode.IsLetter(ch) { return true }
	for _, idCh := range ID_CHARS_SPECIAL {
		if idCh == ch {
			return true
		}
	}
	return false
}
func IsId(ch rune) bool {
    return IsIdFirst(ch) || unicode.IsDigit(ch)
}

type Location struct {
	Source string `json:"source"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
    
    SourceIndex int `json:"-"`
    Raw         int `json:"-"`
}

func (l *Location) Loc() string {
    return fmt.Sprintf("%s:%d:%d", l.Source, l.Line, l.Column)
}

type TokenType uint8
const (
    TokenNone TokenType = iota
    TokenId
    TokenStr
    TokenOParen
    TokenCParen
    TokenOCurly
    TokenCCurly
    TokenOBracket
    TokenCBracket
    TokenComma
    TokenInt
    TokenDouble
    TokenError
)
func (t TokenType) OToC() TokenType {
    switch (t) {
    case TokenOParen:   return TokenCParen
    case TokenOCurly:   return TokenCCurly
    case TokenOBracket: return TokenCBracket
    }
    return TokenNone
}
func (t TokenType) CToO() TokenType {
    switch (t) {
    case TokenCParen:   return TokenOParen
    case TokenCCurly:   return TokenOCurly
    case TokenCBracket: return TokenOBracket
    }
    return TokenNone
}
func (t TokenType) ParenAlter() TokenType {
    switch (t) {
    case TokenOParen:   return TokenCParen
    case TokenCParen:   return TokenOParen
    case TokenOCurly:   return TokenCCurly
    case TokenCCurly:   return TokenOCurly
    case TokenOBracket: return TokenCBracket
    case TokenCBracket: return TokenOBracket
    }
    return TokenNone
}
func (t TokenType) Str() string {
    switch (t) {
    case TokenNone:     return "none"
    case TokenId:       return "id"
    case TokenStr:      return "str"
    case TokenOParen:   return "("
    case TokenCParen:   return ")"
    case TokenOCurly:   return "{"
    case TokenCCurly:   return "}"
    case TokenOBracket: return "["
    case TokenCBracket: return "]"
    case TokenComma:    return ","
    case TokenInt:      return "int"
    case TokenDouble:   return "double"
    case TokenError:    return "error"
    }
    return "unknown"
}

type Source struct {
    Name  string
    Chars []rune
}

type Lexer struct {
    Sources  []Source
    Cursor   Location
    TokenLoc Location
    Type     TokenType
    
    Str      string
    Int      int64
    Double   float64
    Char     rune
    
    Err      error
    ErrLoc   Location
    
    NextFile bool
}
func LexerInit() (l Lexer) {
    l.Cursor.SourceIndex   = -1
    l.TokenLoc.SourceIndex = -1
    l.Type                 = TokenNone
    return
}
func (l *Lexer) AddSourceFile(src string) error {
    bytes, err := os.ReadFile(src)
    if err != nil { return err }
    l.Sources = append(l.Sources, Source{
        Name:  src,
        Chars: []rune(string(bytes)),
    })
    if l.Cursor.SourceIndex == -1 {
        l.Cursor.SourceIndex = 0
        l.Cursor.Source      = src
        l.Cursor.Line        = 1
        l.Cursor.Column      = 1
    }
    return nil
}
func (l *Lexer) AddNamedExpr(name, value string) {
    l.Sources = append(l.Sources, Source{
        Name:  name,
        Chars: []rune(value),
    })
    if l.Cursor.SourceIndex == -1 {
        l.Cursor.SourceIndex = 0
        l.Cursor.Source      = name
        l.Cursor.Line        = 1
        l.Cursor.Column      = 1
    }
}
func (loc *Location) PeekChar(l *Lexer) (ch rune, ok bool) {
    if l.Cursor.SourceIndex == -1 { return }
    if l.Cursor.SourceIndex >= len(l.Sources) { return }
    Chars := l.Sources[loc.SourceIndex].Chars
    if l.Cursor.Raw >= len(Chars) { return }
    return Chars[loc.Raw], true
}
func (loc *Location) SkipChar(l *Lexer, ch rune) (rest bool) {
    Chars := l.Sources[loc.SourceIndex].Chars
    if loc.Raw < len(Chars) {
        if ch == '\n' {
            loc.Line   += 1
            loc.Column  = 1
        } else { loc.Column += 1 }
        loc.Raw  += 1
    }
    if loc.Raw >= len(Chars) {
        if l.Cursor.SourceIndex + 1 >= len(l.Sources) { return }
        l.Cursor.SourceIndex += 1
        l.Cursor.Source = l.Sources[l.Cursor.SourceIndex].Name
        l.Cursor.Line   = 1
        l.Cursor.Column = 1
        l.Cursor.Raw    = 0
        l.NextFile = true
        return true
    }
    l.NextFile = false
    return true
}
func (loc *Location) GetChar(l *Lexer) (ch rune, ok bool) {
    ch, ok = loc.PeekChar(l)
    if !ok { return }
    loc.SkipChar(l, ch)
    return
}
func (l *Lexer) SkipSpaces() (ok bool) {
    for {
        ch, ok := l.Cursor.PeekChar(l)
        if !ok { break }
        if !unicode.IsSpace(ch) { return true }
        l.Cursor.SkipChar(l, ch)
    }
    return
}
func (l *Lexer) SetChToken(ch rune, kind TokenType) {
    l.Cursor.SkipChar(l, ch)
    l.Type = kind
    l.Char = ch
}
func (l *Lexer) SetErr(err error) {
    l.Type = TokenError
    l.Err    = err
    l.ErrLoc = l.TokenLoc
}
func (l *Lexer) UnknownToken(ch rune) {
    l.Cursor.SkipChar(l, ch)
    Ch, _  := log.CharDesc(ch, false)
    l.SetErr(fmt.Errorf("%s does not start any known token", Ch))
}
func (l *Lexer) ParseNumber() bool {
    saved := l.Cursor
    var beforeFloat []rune
    var afterFloat []rune
    var err error
    numStr := ""
    
    ch, ok := l.Cursor.GetChar(l)
    isNegative := ch == '-'
    isFloating := ch == '.'
    
    if !ok { goto restore }
    if unicode.IsDigit(ch) {
        beforeFloat = append(beforeFloat, ch)
    } else if !isNegative && !isFloating { goto restore }
    for {
        ch, ok = l.Cursor.PeekChar(l)
        if !ok { break }
        if ch == '.' {
            if isFloating { goto restore }
            isFloating = true
        } else {
            if !unicode.IsDigit(ch) { break }
            if isFloating {
                afterFloat  = append(afterFloat,  ch)
            } else {
                beforeFloat = append(beforeFloat, ch)
            }
        }
        l.Cursor.SkipChar(l, ch)
        if l.NextFile { break }
    }
    if isFloating && len(afterFloat) == 0 && len(beforeFloat) == 0 {
        goto restore
    }
    if isNegative {
        numStr += "-"
    }
    if len(beforeFloat) == 0 {
        numStr += "0"
    } else {
        numStr += string(beforeFloat)
    }
    if isFloating {
        l.Type = TokenDouble
        numStr += "."
        if len(afterFloat) == 0 {
            numStr += "0"
        } else {
            numStr += string(afterFloat)
        }
        l.Double, err = strconv.ParseFloat(numStr, 64)
    } else {
        l.Type = TokenInt
        l.Int, err = strconv.ParseInt(numStr, 10, 64)
    }
    if err != nil {
        l.SetErr(err)
    }
    return true
restore:
    l.Cursor = saved
    return false
}
func (l *Lexer) ParseId() bool {
    saved := l.Cursor
    ch, ok := l.Cursor.PeekChar(l)
    var chars []rune
    if !ok { goto restore }
    if !IsIdFirst(ch) { goto restore }
    for {
        ch, ok := l.Cursor.PeekChar(l)
        if !ok { break }
        if !IsId(ch) { break }
        chars = append(chars, ch)
        l.Cursor.SkipChar(l, ch)
        if l.NextFile { break }
    }
    if len(chars) == 0 { goto restore }
    l.Type = TokenId
    l.Str  = string(chars)
    return true
restore:
    l.Cursor = saved
    return false
}
func (l *Lexer) Loc() string {
    return l.TokenLoc.Loc()
}
func (l *Lexer) ParseToken() bool {
    ok := l.SkipSpaces()
    if !ok { return false }
    l.TokenLoc = l.Cursor
    ch, _ := l.Cursor.PeekChar(l)
    switch ch {
    case '(':
        l.SetChToken(ch, TokenOParen)
        return true
    case ')':
        l.SetChToken(ch, TokenCParen)
        return true
    case '{':
        l.SetChToken(ch, TokenOCurly)
        return true
    case '}':
        l.SetChToken(ch, TokenCCurly)
        return true
    case '[':
        l.SetChToken(ch, TokenOBracket)
        return true
    case ']':
        l.SetChToken(ch, TokenCBracket)
        return true
    case ',':
        l.SetChToken(ch, TokenComma)
        return true
    case '"':
        l.Cursor.SkipChar(l, ch)
        if l.NextFile {
            l.SetErr(fmt.Errorf("unclosed string literal"))
            return true
        }
        var chars []rune
        escaping := false
        for {
            ch, ok = l.Cursor.PeekChar(l)
            if !ok {
                l.SetErr(fmt.Errorf("unclosed string literal"))
                return true
            }
            l.Cursor.SkipChar(l, ch)
            if !escaping && ch == '"' { break }
            if ch == '\n' || l.NextFile {
                l.SetErr(fmt.Errorf("unclosed string literal"))
                return true
            }
            if escaping {
                switch ch {
                case '"':  fallthrough
                case '\\': break
                case 'r': ch = '\r'
                case 'n': ch = '\n'
                default:
                    l.Type  = TokenError
                    Ch, _  := log.CharDesc(ch, false)
                    l.SetErr(fmt.Errorf("%s unknown escape character", Ch))
                    return true
                }
                escaping = false
            } else if ch == '\\' {
                escaping = true
                continue
            }
            chars = append(chars, ch)
        }
        l.Type = TokenStr
        l.Str = string(chars)
        return true
    default:
        if l.ParseNumber() { return true }
        if l.ParseId()     { return true }
        l.UnknownToken(ch)
        return true
    }
    return false
}
