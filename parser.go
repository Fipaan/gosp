package main

import (
    "github.com/Fipaan/gosp/log"
    "fmt"
    "strconv"
)

func (l *Lexer) PeekToken() (Type TokenType, ok bool) {
    saved := l.Cursor
    ok = l.ParseToken()
    if ok {
        Type = l.Type
    }
    l.Cursor = saved
    return
}
func (l *Lexer) Expect(Type TokenType) error {
    if l.Type != Type {
        return fmt.Errorf("%s: Expected %s, got %s", l.Loc(), Type.Str(), l.Type.Str())
    }
    return nil
}
func (l *Lexer) ParseAndExpect(Type TokenType) error {
    if !l.ParseToken() {
        return fmt.Errorf("%s: Expected %s, got nothing", l.Loc(), Type.Str())
    }
    return l.Expect(Type)
}
func (l *Lexer) ExpectEOF() (err error) {
    if l.NextFile { return }
    if l.SkipSpaces() {
        err = fmt.Errorf("%s: Expected EOF", l.Loc())
    }
    return
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

func (l *Lexer) ParseExpr() (expr Expr, err error) {
    saved := l.Cursor
    ok := l.ParseToken()
    var id string
    var _func *Function = nil
    var t TokenType
    var _expr Expr
    if !ok {
        err = fmt.Errorf("%s: no token found", l.Loc())
        goto restore
    }
    switch l.Type {
        case TokenId:     return Expr{Type: ExprId,     Id:     l.Str},    nil
        case TokenStr:    return Expr{Type: ExprStr,    Str:    l.Str},    nil
        case TokenInt:    return Expr{Type: ExprInt,    Int:    l.Int},    nil
        case TokenDouble: return Expr{Type: ExprDouble, Double: l.Double}, nil
    }
    err = l.Expect(TokenOParen)
    if err != nil { goto restore }
    err = l.ParseAndExpect(TokenId)
    if err != nil { goto restore }
    id = l.Str
    for i := 0; i < len(FUNC_TABLE); i++ {
        FUNC := FUNC_TABLE[i]
        if FUNC.Id == id {
            _func = &FUNC
            break
        }
    }
    if _func == nil {
        err = fmt.Errorf("%s: Unknown function '%s'", l.Loc(), id)
        return
    }
    expr = Expr{Type: ExprFunc, Func: *_func}
    for i := 0; i < len(_func.Types); i++ {
        Type := _func.Types[i]
        switch (Type.QType) {
        case QuantityRegular: log.Todof("QuantityRegular")
        case QuantityAny:
            for {
                t, ok = l.PeekToken()
                if !ok {
                    err = fmt.Errorf("%s: unclosed parens", l.Loc())
                    goto restore
                }
                if t == TokenCParen { break }
                _restore := l.Cursor
                _expr, err = l.ParseExpr()
                if err != nil { goto restore }
                if _expr.Type != Type.Type {
                    l.Cursor = _restore
                    break
                }
                expr.Args = append(expr.Args, _expr)
            }
        case QuantityRange: log.Todof("QuantityRegular")
        }
    }
    _expr, err = l.ParseExpr()
    if err == nil {
        err = fmt.Errorf("%s: invalid types, unexpected %s", l.Loc(), _expr.Type.Str())
        goto restore
    }
    t, ok = l.PeekToken()
    if !ok {
        err = fmt.Errorf("%s: unclosed parens", l.Loc())
        goto restore
    }
    if t != TokenCParen {
        err = fmt.Errorf("%s: unclosed parens %s", l.Loc(), t.Str())
        goto restore
    }
    err = nil
    return
restore:
    l.Cursor = saved
    return
}
