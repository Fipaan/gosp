package main

import (
    "github.com/Fipaan/gosp/log"
    "github.com/Fipaan/gosp/utils"
    "fmt"
    "flag"
    "os"
	"encoding/json"
	"net/http"
)
const (
    NUMBER_1 = 1
    NUMBER_2 = 2.0
)

func parseMain() {
    var depth utils.Stack[TokenType]
    const filename string = "main.go"
    var withPrefix bool   = false
    l := LexerInit()
    err := l.AddSourceFile(filename)
    if err != nil {
        log.Abortf("Couldn't read %s: %s", filename, err.Error())
    }
    for l.ParseToken() {
        if l.Type == TokenNone { return }
        tokenStr := ""
        withPrefix = true
        switch l.Type {
        case TokenOParen: fallthrough
        case TokenOCurly: fallthrough
        case TokenOBracket:
            withPrefix = false
            depth.Push(l.Type)
            tokenStr = fmt.Sprintf("%c", l.Char)
        case TokenCParen: fallthrough
        case TokenCCurly: fallthrough
        case TokenCBracket:
            t, ok := depth.Pop()
            if !ok || t != l.Type.CToO() {
                log.Abortf("%s: unmatched paren", l.Loc())
            }
            withPrefix = false
            tokenStr = fmt.Sprintf("%c", l.Char)
        case TokenComma:
            withPrefix = false
            tokenStr = fmt.Sprintf("%c", l.Char)
        case TokenStr:
            tokenStr = fmt.Sprintf("String(\"%s\")", log.Str2Printable(l.Str))
        case TokenId:
            tokenStr = fmt.Sprintf("Id(%s)",         l.Str)
        case TokenInt:
            tokenStr = fmt.Sprintf("Int(%d)",        l.Int)
        case TokenDouble:
            tokenStr = fmt.Sprintf("Double(%f)",     l.Double)
        case TokenError:
            log.Abortf("%s: %s", l.Loc(), l.Err.Error())
        case TokenNone: fallthrough
        default: log.Unreachable("unknown TokenType")
        }
        if withPrefix {
            log.Printf("\r\n%*s", len(depth)*2, "")
        }
        log.Printf("%s", tokenStr)
    }
    if !withPrefix { log.Printf("\r\n") }
    log.Infof("Successfully read %s!", filename)
}

func fileExample() {
    _filename := flag.String("filename", "main.go", "path to file to be parsed")
    flag.Parse()
    filename := *_filename
    l := LexerInit()
    err := l.AddSourceFile(filename)
    if err != nil {
        log.Abortf("Couldn't read %s: %s", filename, err.Error())
    }
    var expr Expr
    expr, err = l.ParseExpr()
    if err != nil {
        log.Errorf("%s", err.Error())
        os.Exit(1)
    }
    log.Printf("%s", expr.Eval())
}

type ExprRequest struct {
	Expr string `json:"expr"`
}

type ExprResponse struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func exprHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExprRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ExprResponse{Error: "invalid JSON"})
		return
	}

	if req.Expr == "" {
		writeJSON(w, ExprResponse{Error: "expr is required"})
		return
	}
    
    l := LexerInit()
    l.AddNamedExpr("post-request", req.Expr)
    expr, err := l.ParseExpr()
    if err != nil {
		writeJSON(w, ExprResponse{Error: err.Error()})
		return
    }
	writeJSON(w, ExprResponse{Result: expr.Eval()})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func main() {
	http.HandleFunc("/api/expr", exprHandler)

    const PORT = ":8000"
	log.Infof("listening on %s", PORT)
	http.ListenAndServe(PORT, nil)
}
