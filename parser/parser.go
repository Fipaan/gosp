package parser

import (
    "github.com/Fipaan/gosp/log"
    "github.com/Fipaan/gosp/lexer"
    "fmt"
    "strconv"
)

type Parser struct {
    lexer.Lexer
}

func ParserInit() Parser {
    return Parser{lexer.LexerInit()}
}

func (p *Parser) PeekToken() (Type lexer.TokenType, ok bool) {
    saved := p.Cursor
    ok = p.ParseToken()
    if ok {
        Type = p.Type
    }
    p.Cursor = saved
    return
}
func (p *Parser) Expect(Type lexer.TokenType) bool {
    if p.Type != Type {
        p.SetErr(fmt.Errorf("Expected %s, got %s", Type.Str(), p.Type.Str()))
        return false
    }
    return true
}
func (p *Parser) ParseAndExpect(Type lexer.TokenType) bool {
    if !p.ParseToken() {
        p.SetErr(fmt.Errorf("Expected %s, got nothing", Type.Str()))
        return false
    }
    return p.Expect(Type)
}
func (p *Parser) ExpectEOF() bool {
    if p.NextFile { return true }
    if p.SkipSpaces() {
        p.SetErr(fmt.Errorf("%s: Expected EOF", p.Loc()))
        return false
    }
    return true
}

type ExprType uint8
const (
    ExprFunc ExprType = iota
    ExprId
    ExprStr
    ExprInt
    ExprDouble
)
func (t ExprType) Str() string {
    switch (t) {
    case ExprFunc:   return "function"
    case ExprId:     return "id"
    case ExprStr:    return "str"
    case ExprInt:    return "int"
    case ExprDouble: return "double"
    }
    return "unknown"
}
type Expr struct {
    Type   ExprType
    Func   Function
    Args   []Expr
    Id     string
    Str    string
    Int    int64
    Double float64
}
func (expr *Expr) Eval() string {
    switch (expr.Type) {
    case ExprFunc:
        res := expr.Func.Impl(expr.Args)
        return (&res).Eval()
    case ExprId:  return expr.Id
    case ExprStr: return expr.Str
    case ExprInt:
        return strconv.FormatInt(expr.Int, 10)
    case ExprDouble:
        return fmt.Sprintf("%f", expr.Double)
    }
    log.Abortf("unknown type")
    return ""
}
type QuantityType uint8
const (
    QuantityRegular QuantityType = iota
    QuantityAny
    QuantityRange
)
type FunctionType struct {
    Type  ExprType
    QType QuantityType
    To    uint
    From  uint
}
type Function struct {
    Id    string
    Types []FunctionType
    Impl  func([]Expr) Expr
}
var FUNC_TABLE = []Function {
    Function{
        Id: "+",
        Types: []FunctionType{
            FunctionType{Type: ExprDouble, QType: QuantityAny},
        },
        Impl: func(args []Expr) Expr {
            result := 0.0
            for i := 0; i < len(args); i++ {
                result += args[i].Double 
            }
            return Expr{Type: ExprDouble, Double: result}
        },
    },
}

func (p *Parser) ParseExpr() (expr Expr, ok bool) {
    saved := p.Cursor
    ok = p.ParseToken()
    var id string
    var _func *Function = nil
    var t lexer.TokenType
    var _expr Expr
    if !ok {
        p.SetErr(fmt.Errorf("no token found"))
        goto restore
    }
    switch p.Type {
        case lexer.TokenId:     return Expr{Type: ExprId,     Id:     p.Str},    true
        case lexer.TokenStr:    return Expr{Type: ExprStr,    Str:    p.Str},    true
        case lexer.TokenInt:    return Expr{Type: ExprInt,    Int:    p.Int},    true
        case lexer.TokenDouble: return Expr{Type: ExprDouble, Double: p.Double}, true
    }
    ok = p.Expect(lexer.TokenOParen)
    if !ok { goto restore }
    ok = p.ParseAndExpect(lexer.TokenId)
    if !ok { goto restore }
    id = p.Str
    for i := 0; i < len(FUNC_TABLE); i++ {
        FUNC := FUNC_TABLE[i]
        if FUNC.Id == id {
            _func = &FUNC
            break
        }
    }
    if _func == nil {
        p.SetErr(fmt.Errorf("Unknown function '%s'", id))
        ok = false
        return
    }
    expr = Expr{Type: ExprFunc, Func: *_func}
    for i := 0; i < len(_func.Types); i++ {
        Type := _func.Types[i]
        switch (Type.QType) {
        case QuantityRegular: log.Todof("QuantityRegular")
        case QuantityAny:
            for {
                t, ok = p.PeekToken()
                if !ok {
                    p.SetErr(fmt.Errorf("unclosed parens"))
                    goto restore
                }
                if t == lexer.TokenCParen { break }
                _restore := p.Cursor
                _expr, ok = p.ParseExpr()
                if !ok { goto restore }
                if _expr.Type != Type.Type {
                    p.Cursor = _restore
                    break
                }
                expr.Args = append(expr.Args, _expr)
            }
        case QuantityRange: log.Todof("QuantityRegular")
        }
    }
    _expr, ok = p.ParseExpr()
    if ok {
        ok = false
        p.SetErr(fmt.Errorf("invalid types, unexpected %s", _expr.Type.Str()))
        goto restore
    }
    t, ok = p.PeekToken()
    if !ok {
        p.SetErr(fmt.Errorf("unclosed parens"))
        goto restore
    }
    if t != lexer.TokenCParen {
        ok = false
        p.SetErr(fmt.Errorf("unclosed parens %s", t.Str()))
        goto restore
    }
    return
restore:
    p.Cursor = saved
    return
}
