package parser

import (
    "github.com/Fipaan/gosp/log"
    "github.com/Fipaan/gosp/lexer"
    "fmt"
)

type Parser struct {
    lexer.Lexer
}

func ParserInit() Parser {
    return Parser{lexer.LexerInit()}
}

func (p *Parser) GetToken() (Type lexer.TokenType, ok bool) {
    ok = p.ParseToken()
    if ok { Type = p.Type }
    return
}
func (p *Parser) PeekToken() (Type lexer.TokenType, ok bool) {
    saved := p.Cursor
    Type, ok = p.GetToken()
    p.Cursor = saved
    return
}
func (p *Parser) ExpectedErr(expected, got string) {
    p.SetErr(fmt.Errorf("Expected %s, got %s", expected, got))
}
func (p *Parser) PExpectedErr(prefix, expected, got string) {
    p.SetErr(fmt.Errorf("%s: Expected %s, got %s", prefix, expected, got))
}
func (p *Parser) Expect(Type lexer.TokenType) bool {
    if p.Type != Type {
        p.ExpectedErr(Type.Str(), p.Type.Str())
        return false
    }
    return true
}
func (p *Parser) ParseAndExpect(Type lexer.TokenType) bool {
    if !p.ParseToken() {
        p.ExpectedErr(Type.Str(), "nothing")
        return false
    }
    return p.Expect(Type)
}
func (p *Parser) ExpectEOF() bool {
    if p.NextFile { return true }
    if p.SkipSpaces(false) == lexer.ReadOk {
        p.ExpectedErr("EOF", "something else")
        return false
    }
    return true
}

type ExprType uint8
const (
    ExprNone ExprType = iota
    ExprFunc
    ExprId
    ExprStr
    ExprInt
    ExprDouble
)
func (t ExprType) Str() string {
    switch (t) {
    case ExprNone:   return "none"
    case ExprFunc:   return "function"
    case ExprId:     return "id"
    case ExprStr:    return "str"
    case ExprInt:    return "int"
    case ExprDouble: return "double"
    }
    return "unknown"
}
func Token2ExprType(t lexer.TokenType) ExprType {
    switch t {
        case lexer.TokenId:     return ExprId
        case lexer.TokenStr:    return ExprStr
        case lexer.TokenInt:    return ExprInt
        case lexer.TokenDouble: return ExprDouble
    }
    return ExprNone
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
func (p *Parser) Token2Expr(Type lexer.TokenType) (Expr, bool) {
    expr := Expr{Type: Token2ExprType(Type)}
    if expr.Type == ExprNone { return expr, false }
    switch expr.Type {
        case ExprId:     expr.Id     = p.Str
        case ExprStr:    expr.Str    = p.Str
        case ExprInt:    expr.Int    = p.Int
        case ExprDouble: expr.Double = p.Double
        default: log.Unreachable("unexpected expression type: %s", expr.Type.Str())
    }
    return expr, true
}
func (p *Parser) Token2ExprCurr() (expr Expr, ok bool) {
    return p.Token2Expr(p.Type)
}
func (expr *Expr) Eval() string {
    switch (expr.Type) {
    case ExprNone: return "undefined"
    case ExprFunc:
        res := expr.Func.Impl(expr.Args)
        return (&res).Eval()
    case ExprId:  return expr.Id
    case ExprStr: return expr.Str
    case ExprInt:
        return fmt.Sprintf("%d", expr.Int)
    case ExprDouble:
        return fmt.Sprintf("%f", expr.Double)
    }
    log.Abortf("unknown type"); return ""
}
type ArgType struct {
    Type  ExprType
}
func (at ArgType) Str() string {
    return fmt.Sprintf("%s argument", at.Type.Str())
}
type Function struct {
    Id    string
    Types []ArgType
    Impl  func([]Expr) Expr
    VType ArgType // variadic args
}
func (p *Parser) ExpectedFuncErr(Func Function, expected, got string) {
    p.PExpectedErr(Func.Id, expected, got)
}
func (p *Parser) ExpectedFuncNotNothing(Func Function, AType ArgType) {
    p.ExpectedFuncErr(Func, AType.Str(), "nothing")
}
func (p *Parser) ExpectedFuncMismatch(Func Function, AType ArgType, expr Expr) {
    p.ExpectedFuncErr(Func, AType.Str(), expr.Type.Str())
}
func (p *Parser) ExpectedFuncNotEnough(Func Function, expected string) {
    p.SetErr(fmt.Errorf("%s: Not enough arguments (expected %s)", Func.Id, expected))
}
func (p *Parser) ExpectedFuncTooMany(Func Function, expr Expr) {
    p.SetErr(fmt.Errorf("%s: Too many arguments (unexpected %s)", Func.Id, expr.Type.Str()))
}
func (p *Parser) ExpectedFuncEnd(Func Function, got string) {
    p.ExpectedFuncErr(Func, lexer.TokenCParen.Str(), got)
}
var FUNC_TABLE = []Function {
    Function{
        Id: "+",
        VType: ArgType{Type: ExprDouble},
        Impl: func(args []Expr) Expr {
            result := 0.0
            for i := 0; i < len(args); i++ {
                result += args[i].Double 
            }
            return Expr{Type: ExprDouble, Double: result}
        },
    },
}
func (p *Parser) ParseFuncArgAny(Func Function, strType string,
                                 isVType bool) (expr Expr, ok bool) {
    var ttype lexer.TokenType
    ttype, ok = p.PeekToken()
    if !ok {
        p.ExpectedFuncErr(Func, strType, "nothing")
        return
    }
    if ttype == lexer.TokenCParen {
        if !isVType {
            p.ExpectedFuncNotEnough(Func, strType)
            ok = false
        }
        return
    }
    return p.ParseExpr()
}
func (p *Parser) ParseFuncArg(Func Function, AType ArgType,
                              isVType bool) (expr Expr, ok bool) {
    saved := p.Cursor
    expr, ok = p.ParseFuncArgAny(Func, AType.Str(), isVType)
    if !ok { goto restore }
    if isVType && expr.Type == ExprNone { return }
    if expr.Type != AType.Type {
        p.ExpectedFuncMismatch(Func, AType, expr)
        ok = false
        goto restore
    }
    return
restore:
    p.Cursor = saved
    return
}
func (p *Parser) ParseFunc() (expr Expr, ok bool) {
    var ttype    lexer.TokenType
    var Func    *Function
    var id       string
    var exprArg  Expr
    saved := p.Cursor
    ok = p.ParseAndExpect(lexer.TokenOParen)
    if !ok { goto restore }
    ok  = p.ParseAndExpect(lexer.TokenId)
    if !ok { goto restore }
    id = p.Str
    for i := 0; i < len(FUNC_TABLE); i++ {
        FUNC := FUNC_TABLE[i]
        if FUNC.Id == id {
            Func = &FUNC
            break
        }
    }
    if Func == nil {
        p.SetErr(fmt.Errorf("Unknown function '%s'", id))
        ok = false
        return
    }
    expr = Expr{Type: ExprFunc, Func: *Func}
    for i := 0; i < len(expr.Func.Types); i++ {
        AType := expr.Func.Types[i]
        const isVType = false
        exprArg, ok = p.ParseFuncArg(expr.Func, AType, isVType)
        if !ok { goto restore }
        expr.Args = append(expr.Args, exprArg)
    }
    if expr.Func.VType.Type != ExprNone {
        AType := expr.Func.VType
        for {
            const isVType = true
            exprArg, ok = p.ParseFuncArg(expr.Func, AType, isVType)
            if !ok { goto restore }
            if exprArg.Type == ExprNone { break }
            expr.Args = append(expr.Args, exprArg)
        }
    }
    const isVType = true
    exprArg, ok = p.ParseFuncArgAny(expr.Func, "", isVType)
    if ok && exprArg.Type != ExprNone {
        p.ExpectedFuncTooMany(expr.Func, exprArg)
        ok = false
        goto restore
    }
    ttype, ok = p.PeekToken()
    if !ok {
        p.ExpectedFuncEnd(expr.Func, "nothing")
        goto restore
    }
    if ttype != lexer.TokenCParen {
        p.ExpectedFuncEnd(expr.Func, ttype.Str())
        ok = false
        goto restore
    }
    return
restore:
    p.Cursor = saved
    return
}

func (p *Parser) ParseExpr() (expr Expr, ok bool) {
    saved := p.Cursor
    var ttype lexer.TokenType
    ttype, ok = p.PeekToken()
    if !ok { return Expr{Type: ExprNone}, true }
    expr, ok = p.Token2Expr(ttype)
    if ok {
        _, ok = p.GetToken()
        return
    }
    if ttype == lexer.TokenOParen {
        expr, ok = p.ParseFunc()
        if !ok { goto restore }
        return
    }
    p.SetErr(fmt.Errorf("Unknown token: %s", ttype.Str()))
restore:
    p.Cursor = saved
    return
}
