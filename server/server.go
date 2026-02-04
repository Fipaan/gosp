package server

import (
    "context"
	"net/http"
	"strings"
	"time"
    
    "github.com/Fipaan/gosp/parser"
)

type Server struct {
    DB          *Storage
    CookieName   string
    AuthTTL      time.Duration
    Addr         string
}

func (sv *Server) ExtractAuthKey(r *http.Request) string {
	if c, err := r.Cookie(sv.CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	authz := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	if v := strings.TrimSpace(r.Header.Get("X-Auth-Key")); v != "" {
		return v
	}
	return ""
}

func (sv *Server) SetAuthCookie(w http.ResponseWriter, authKey string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sv.CookieName,
		Value:    authKey,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
		// Secure: true, // enable behind HTTPS
	})
}

func (sv *Server) ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sv.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (sv *Server) WithTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	// Basic request-scoped DB timeout
	return context.WithTimeout(r.Context(), 5*time.Second)
}

func (sv *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, nil, "Method not allowed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !ReadJSONBody(w, r, &req) {
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		WriteAPIError(w, http.StatusBadRequest, nil, "username and password are required")
		return
	}
	if len(req.Username) > 64 {
		WriteAPIError(w, http.StatusBadRequest, nil, "username too long")
		return
	}

	ctx, cancel := sv.WithTimeout(r)
	defer cancel()

	if err := sv.DB.CreateUser(ctx, req.Username, req.Password); err != nil {
		if err.Error() == "username already exists" {
			WriteAPIError(w, http.StatusConflict, nil, err.Error())
			return
		}
		WriteAPIError(w, http.StatusInternalServerError, nil, "database error")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func (sv *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, nil, "Method not allowed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !ReadJSONBody(w, r, &req) {
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		WriteAPIError(w, http.StatusBadRequest, nil, "username and password are required")
		return
	}

	ctx, cancel := sv.WithTimeout(r)
	defer cancel()

	ok, err := sv.DB.VerifyUser(ctx, req.Username, req.Password)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, nil, "database error")
		return
	}
	if !ok {
		WriteAPIError(w, http.StatusUnauthorized, nil, "invalid credentials")
		return
	}

	authKey, exp, err := sv.DB.CreateSession(ctx, req.Username, sv.AuthTTL)
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, nil, "database error")
		return
	}

	sv.SetAuthCookie(w, authKey, exp)
	w.Header().Set("X-Auth-Key", authKey) // optional for non-browser clients

	WriteJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func (sv *Server) RequireAuth(next func(w http.ResponseWriter, r *http.Request, sess SessionDoc)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authKey := sv.ExtractAuthKey(r)
		if authKey == "" {
			WriteAPIError(w, http.StatusUnauthorized, nil, "authKey is required")
			return
		}

		ctx, cancel := sv.WithTimeout(r)
		defer cancel()

		sess, ok, err := sv.DB.TouchSession(ctx, authKey)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, nil, "database error")
			return
		}
		if !ok {
			WriteAPIError(w, http.StatusUnauthorized, nil, "invalid or expired authKey")
			return
		}
		next(w, r, sess)
	}
}

func (sv *Server) HandleLogout(w http.ResponseWriter, r *http.Request, sess SessionDoc) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, nil, "Method not allowed")
		return
	}

	authKey := sv.ExtractAuthKey(r)
	if authKey != "" {
		ctx, cancel := sv.WithTimeout(r)
		defer cancel()
		_ = sv.DB.DeleteSession(ctx, authKey)
	}

	sv.ClearAuthCookie(w)
	WriteJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func (sv *Server) HandleHistory(w http.ResponseWriter, r *http.Request, sess SessionDoc) {
	if r.Method != http.MethodGet {
		WriteAPIError(w, http.StatusMethodNotAllowed, nil, "Method not allowed")
		return
	}

	ctx, cancel := sv.WithTimeout(r)
	defer cancel()

	items, err := sv.DB.GetHistory(ctx, sess.Username, 50) // placeholder limit
	if err != nil {
		WriteAPIError(w, http.StatusInternalServerError, nil, "database error")
		return
	}

	// If you must return ONLY {"status":"OK"}, remove "items".
	WriteJSON(w, http.StatusOK, map[string]any{
		"status": "OK",
		"items":  items,
	})
}

func (sv *Server) HandleExpr(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteAPIError(w, http.StatusMethodNotAllowed, nil, "Method not allowed")
		return
	}

	var req struct {
		Expr string `json:"expr"`
	}
	if !ReadJSONBody(w, r, &req) {
		return
	}

	// authKey optional here
	var username string
	if ak := sv.ExtractAuthKey(r); ak != "" {
		ctx, cancel := sv.WithTimeout(r)
		defer cancel()

		if sess, ok, err := sv.DB.TouchSession(ctx, ak); err == nil && ok {
			username = sess.Username
		}
	}
    
    p := parser.ParserInit()
    p.AddSourceNamed("post-request", req.Expr)
    expr, ok := p.ParseExpr()
    if !ok {
		WriteAPIError(w, http.StatusBadRequest, &p.ErrLoc, p.Err.Error())
		return
    }
    res := expr.Eval()

	if username != "" {
		ctx, cancel := sv.WithTimeout(r)
		defer cancel()
		sv.DB.AppendHistory(ctx, username, req.Expr, res)
	}

	WriteJSON(w, http.StatusOK, map[string]string{"result": res})
}
