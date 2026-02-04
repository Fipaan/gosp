package server

import (
	"strings"

	"github.com/Fipaan/gosp/lexer"
	"github.com/Fipaan/gosp/parser"
)

// parses/evals multiple expressions from all sources
// returns a transcript string
// firstErrLoc nil on full success
func EvalTS(p *parser.Parser, gs *parser.GospState) (out string, firstErrLoc *lexer.Location) {
	var b strings.Builder

	var haveFirstErr bool

    sourceIndex := p.Cursor.SourceIndex
	fullText := string(p.Sources[sourceIndex].Chars)
	lines := strings.Split(fullText, "\n")

	for {
		if p.SkipSpaces(true) != lexer.ReadOk {
			break
		}
        if sourceIndex != p.Cursor.SourceIndex {
            sourceIndex = p.Cursor.SourceIndex
	        fullText = string(p.Sources[sourceIndex].Chars)
	        lines = strings.Split(fullText, "\n")
        }

		locStart := p.Cursor
		expr, ok := p.ParseExpr(gs)
		locEnd := p.Cursor

		if ok {
			frag := p.TokenStr(locStart, locEnd)
			b.WriteString("`")
			b.WriteString(frag)
			b.WriteString("` ->\n")
			b.WriteString("Result: ")
			b.WriteString(expr.ToStr(gs))
			b.WriteString("\n")
			continue
		}

		if !haveFirstErr {
			loc := p.ErrLoc
			firstErrLoc = &loc
			haveFirstErr = true
		}

		loc := p.ErrLoc
		if loc.Line >= 1 && int(loc.Line) <= len(lines) {
			line := lines[loc.Line-1]
			b.WriteString(line)
			b.WriteString("\n")

			if loc.Column < 1 {
				loc.Column = 1
			}
			b.WriteString(strings.Repeat(" ", loc.Column-1))
			b.WriteString("^")

			_ = p.SkipExpr()
			locEnd = p.Cursor
			tStr := p.TokenStr(loc, locEnd)

			if len(tStr) > 1 {
				b.WriteString(strings.Repeat("~", len(tStr)-1))
			}
			b.WriteString("\n")
		}

		b.WriteString(loc.Loc())
		b.WriteString(": ")
		if p.Err != nil {
			b.WriteString(p.Err.Error())
		} else {
			b.WriteString("unknown error")
		}
		b.WriteString("\n")
	}

	return b.String(), firstErrLoc
}
