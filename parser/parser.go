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

type ExprKind uint8
const (
    ExprNone ExprKind = iota
    ExprFunc
    ExprId
    ExprStr
    ExprInt
    ExprDouble
)
func (t ExprKind) Str() string {
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
func Token2ExprKind(t lexer.TokenType) ExprKind {
    switch t {
        case lexer.TokenId:     return ExprId
        case lexer.TokenStr:    return ExprStr
        case lexer.TokenInt:    return ExprInt
        case lexer.TokenDouble: return ExprDouble
    }
    return ExprNone
}
type Expr struct {
    Kind   ExprKind
    Func   Function
    Args   []Expr
    Id     string
    Str    string
    Int    int64
    Double float64
}
func (p *Parser) Token2Expr(Type lexer.TokenType) (Expr, bool) {
    expr := Expr{Kind: Token2ExprKind(Type)}
    if expr.Kind == ExprNone { return expr, false }
    switch expr.Kind {
        case ExprId:     expr.Id     = p.Str
        case ExprStr:    expr.Str    = p.Str
        case ExprInt:    expr.Int    = p.Int
        case ExprDouble: expr.Double = p.Double
        default: log.Unreachable("unexpected expression type: %s", expr.Kind.Str())
    }
    return expr, true
}
func (p *Parser) Token2ExprCurr() (expr Expr, ok bool) {
    return p.Token2Expr(p.Type)
}
func (expr *Expr) GetExprType() ExprType {
    EType := ExprType{Kind: expr.Kind}
    switch (EType.Kind) {
    case ExprFunc:
        EType.Func = expr.Func.Type
    case ExprNone:   fallthrough
    case ExprId:     fallthrough
    case ExprStr:    fallthrough
    case ExprInt:    fallthrough
    case ExprDouble: break
    default: log.Unreachable("unknown expr type: %s", EType.Kind.Str())
    }
    return EType
}
func (expr *Expr) Eval(gs *GospState) Expr {
    rexpr := *expr
    switch (expr.Kind) {
    case ExprFunc:   rexpr = expr.Func.Impl(gs, expr.Args)
    case ExprNone:   fallthrough
    case ExprId:     fallthrough
    case ExprStr:    fallthrough
    case ExprInt:    fallthrough
    case ExprDouble: break
    default: log.Unreachable("unknown expr type: %s", expr.Kind.Str())
    }
    return rexpr
}
func (expr *Expr) ToStr(gs *GospState) string {
    rexpr := expr.Eval(gs)
    switch (rexpr.Kind) {
    case ExprNone: return "undefined"
    case ExprFunc:
        log.Unreachable("unexpected expr type: %s", rexpr.Kind.Str())
    case ExprId:  return rexpr.Id
    case ExprStr: return rexpr.Str
    case ExprInt:
        return fmt.Sprintf("%d", rexpr.Int)
    case ExprDouble:
        return fmt.Sprintf("%f", rexpr.Double)
    default: log.Unreachable("unknown expr type: %s", rexpr.Kind.Str())
    }
 return ""
}
type FuncType struct {
    Types []ExprType
    VType *ExprType
    RType *ExprType
}
type ExprType struct {
    Kind ExprKind
    Func FuncType
}
func (et ExprType) Str() string {
    return fmt.Sprintf("%s argument", et.Kind.Str())
}
type Function struct {
    Id    string
    Type  FuncType
    Impl  func(*GospState, []Expr) Expr
}
type Binding struct {
    Id  string
    Val Expr
}
type GospState struct {
    Funcs    []Function
    Bindings []Binding
}
func GospInit() GospState {
    return GospState {
        Funcs:[]Function {
            Function{
                Id: "+",
                Type: FuncType{
                    VType: &ExprType{Kind: ExprDouble},
                    RType: &ExprType{Kind: ExprDouble},
                },
                Impl: func(gs *GospState, args []Expr) Expr {
                    result := 0.0
                    for i := 0; i < len(args); i++ {
                        result += args[i].Eval(gs).Double 
                    }
                    return Expr{Kind: ExprDouble, Double: result}
                },
            },
        },
    }
}
func (p *Parser) ExpectedFuncErr(Func Function, expected, got string) {
    p.PExpectedErr(Func.Id, expected, got)
}
func (p *Parser) ExpectedFuncNotNothing(Func Function, EType ExprType) {
    p.ExpectedFuncErr(Func, EType.Str(), "nothing")
}
func (p *Parser) ExpectedFuncMismatch(Func Function, EType ExprType, expr Expr) {
    p.ExpectedFuncErr(Func, EType.Str(), expr.GetExprType().Str())
}
func (p *Parser) ExpectedFuncNotEnough(Func Function, expected string) {
    p.SetErr(fmt.Errorf("%s: Not enough arguments (expected %s)", Func.Id, expected))
}
func (p *Parser) ExpectedFuncTooMany(Func Function, expr Expr) {
    p.SetErr(fmt.Errorf("%s: Too many arguments (unexpected %s)", Func.Id, expr.Kind.Str()))
}
func (p *Parser) ExpectedFuncEnd(Func Function, got string) {
    p.ExpectedFuncErr(Func, lexer.TokenCParen.Str(), got)
}
func (p *Parser) ParseFuncArgAny(gs *GospState, Func Function, strType string,
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
    return p.ParseExpr(gs)
}
func (et ExprType) SimpType() ExprType {
    thisType := ExprType{Kind: et.Kind}
    switch (thisType.Kind) {
    case ExprFunc:
        if et.Func.RType != nil {
            thisType = *et.Func.RType
        } else {
            thisType.Kind = ExprNone
        }
    case ExprNone:   fallthrough
    case ExprId:     fallthrough
    case ExprStr:    fallthrough
    case ExprInt:    fallthrough
    case ExprDouble: break
    default: log.Unreachable("unknown expr type: %s", thisType.Kind.Str())
    }
    return thisType
}
func (et ExprType) SameType(other ExprType) (ok bool) {
    if et.Kind != other.Kind { return }
    switch (et.Kind) {
    case ExprFunc:   fallthrough // TODO: proper function check
    case ExprNone:   fallthrough
    case ExprId:     fallthrough
    case ExprStr:    fallthrough
    case ExprInt:    fallthrough
    case ExprDouble: break
    default: log.Unreachable("unknown expr type: %s", et.Kind.Str())
    }
    return true
}
func (p *Parser) ParseFuncArg(gs *GospState, Func Function, EType ExprType,
                              isVType bool) (expr Expr, ok bool) {
    savedCur := p.Cursor
    savedBindings := gs.Bindings
    expr, ok = p.ParseFuncArgAny(gs, Func, EType.Str(), isVType)
    if !ok { goto restore }
    for expr.Kind == ExprId {
        var exprBind *Binding
        for i := 0; i < len(gs.Bindings); i++ {
            bind := gs.Bindings[i]
            if bind.Id == expr.Id {
                exprBind = &bind
                break
            }
        }
        if exprBind == nil { break }
        expr = exprBind.Val
    }
    if isVType && expr.Kind == ExprNone { return }
    if !EType.SameType(expr.GetExprType().SimpType()) {
        p.ExpectedFuncMismatch(Func, EType, expr)
        ok = false
        goto restore
    }
    return
restore:
    p.Cursor = savedCur
    gs.Bindings = savedBindings
    return
}
func (p *Parser) ParseLet(gs *GospState) (expr Expr, ok, validObj bool) {
    var binding string
    var bindingVal Expr
    savedCur := p.Cursor
    savedBindings := gs.Bindings
    ok = p.ParseAndExpect(lexer.TokenOParen)
    if !ok { goto restore }
    ok  = p.ParseAndExpect(lexer.TokenId)
    if !ok { goto restore }
    if p.Str != "let" {
        p.ExpectedErr("let", p.Str)
        ok = false
        goto restore
    }
    validObj = true
    ok = p.ParseAndExpect(lexer.TokenId)
    if !ok { goto restore }
    binding = p.Str
    for i := 0; i < len(gs.Funcs); i++ {
        if gs.Funcs[i].Id == binding {
            p.SetErr(fmt.Errorf("`%s` already exists: function", binding))
            ok = false
            goto restore
        }
    }
    for i := 0; i < len(gs.Bindings); i++ {
        if gs.Bindings[i].Id == binding {
            p.SetErr(fmt.Errorf("`%s` already exists: let", binding))
            ok = false
            goto restore
        }
    }
    bindingVal, ok = p.ParseExpr(gs)
    if !ok { goto restore }
    bindingVal = bindingVal.Eval(gs)
    gs.Bindings = append(gs.Bindings, Binding{Id: binding, Val: bindingVal})
    expr, ok = p.ParseExpr(gs)
    if !ok { goto restore }
    ok = p.ParseAndExpect(lexer.TokenCParen)
    if !ok { goto restore }
    gs.Bindings = savedBindings
    return
restore:
    p.Cursor = savedCur
    gs.Bindings = savedBindings
    return
}
func (p *Parser) ParseDefun(gs *GospState) (expr Expr, ok, validObj bool) {
    // var Func    *Function
    // var id       string
    // var exprArg  Expr
    savedCur := p.Cursor
    savedBindings := gs.Bindings
    ok = p.ParseAndExpect(lexer.TokenOParen)
    if !ok { goto restore }
    ok  = p.ParseAndExpect(lexer.TokenId)
    if !ok { goto restore }
    if p.Str != "defun" {
        p.ExpectedErr("defun", p.Str)
        ok = false
        goto restore
    }
    validObj = true
    log.Todof("defun")
    return
restore:
    p.Cursor = savedCur
    gs.Bindings = savedBindings
    return
}
func (p *Parser) ParseFunc(gs *GospState) (expr Expr, ok bool) {
    var Func    *Function
    var id       string
    var exprArg  Expr
    savedCur := p.Cursor
    savedBindings := gs.Bindings
    ok = p.ParseAndExpect(lexer.TokenOParen)
    if !ok { goto restore }
    ok  = p.ParseAndExpect(lexer.TokenId)
    if !ok { goto restore }
    id = p.Str
    for i := 0; i < len(gs.Funcs); i++ {
        FUNC := gs.Funcs[i]
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
    expr = Expr{Kind: ExprFunc, Func: *Func}
    for i := 0; i < len(expr.Func.Type.Types); i++ {
        EType := expr.Func.Type.Types[i]
        const isVType = false
        exprArg, ok = p.ParseFuncArg(gs, expr.Func, EType, isVType)
        if !ok { goto restore }
        expr.Args = append(expr.Args, exprArg)
    }
    const isVType = true
    if expr.Func.Type.VType != nil {
        EType := *expr.Func.Type.VType
        for {
            exprArg, ok = p.ParseFuncArg(gs, expr.Func, EType, isVType)
            if !ok { goto restore }
            if exprArg.Kind == ExprNone { break }
            expr.Args = append(expr.Args, exprArg)
        }
    }
    exprArg, ok = p.ParseFuncArgAny(gs, expr.Func, "", isVType)
    if ok && exprArg.Kind != ExprNone {
        p.ExpectedFuncTooMany(expr.Func, exprArg)
        ok = false
        goto restore
    }
    ok = p.ParseAndExpect(lexer.TokenCParen)
    if !ok {
        goto restore
    }
    return
restore:
    p.Cursor = savedCur
    gs.Bindings = savedBindings
    return
}

func (p *Parser) ParseExpr(gs *GospState) (expr Expr, ok bool) {
    savedCur := p.Cursor
    savedBindings := gs.Bindings
    var ttype lexer.TokenType
    ttype, ok = p.PeekToken()
    if !ok { return Expr{Kind: ExprNone}, true }
    expr, ok = p.Token2Expr(ttype)
    if ok {
        _, ok = p.GetToken()
        return
    }
    if ttype == lexer.TokenOParen {
        var validObj bool
        expr, ok, validObj = p.ParseLet(gs)
        if ok { return }
        if validObj { goto restore }
        expr, ok, validObj = p.ParseDefun(gs)
        if ok { return }
        if validObj { goto restore }
        expr, ok = p.ParseFunc(gs)
        if !ok { goto restore }
        return
    }
    p.SetErr(fmt.Errorf("Unknown token: %s", ttype.Str()))
restore:
    p.Cursor = savedCur
    gs.Bindings = savedBindings
    return
}

func (p *Parser) SkipExpr() bool {
    something := false
    exprDepth := 0
    sourceIndex := p.Cursor.SourceIndex
    line := p.Cursor.Line
    for {
        ok := p.ParseToken()
        if !ok { return something }
        if p.Cursor.Line != line ||
           p.Cursor.SourceIndex != sourceIndex { return something }
        something = true
        if p.Type == lexer.TokenOParen {
            exprDepth += 1
            continue
        }
        if p.Type == lexer.TokenCParen {
            exprDepth -= 1
            if exprDepth <= 0 { return something }
        }
    }
}
