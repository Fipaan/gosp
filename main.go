package main

import (
    "context"
	"net/http"
    "os"
	"time"

    "github.com/Fipaan/gosp/server"
    "github.com/Fipaan/gosp/log"
)

func ensureEnv(name string) string {
    val := os.Getenv(name)
    if val == "" { log.Abortf("%s was not set", name) }
    return val
}

func initDB(ctx context.Context) (server.Storage, func(context.Context) error){
    mongoURI := ensureEnv("MONGO_URI")
	dbName   := ensureEnv("MONGO_DB")
    secret   := ensureEnv("AUTH_HMAC_SECRET")

	db, closeFn, err := server.NewStore(ctx, mongoURI, dbName, []byte(secret))
	if err != nil {
        log.Abortf("Couldn't initialize db: %s", err.Error())
	}
    return db, closeFn
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

    db, closeFn := initDB(ctx)
	
	defer func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer ccancel()
		_ = closeFn(cctx)
	}()

	sv := &server.Server{
        DB:         &db,
	    CookieName: "authKey",
        AuthTTL:    30 * 24 * time.Hour,
	    Addr:       ":8000",
    }

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", sv.HandleRegister)
	mux.HandleFunc("/api/login", sv.HandleLogin)
	mux.HandleFunc("/api/expr", sv.HandleExpr)

    mux.Handle("/",
        http.StripPrefix("/",
            http.FileServer(http.Dir("./public")),
        ),
    )

	mux.HandleFunc("/api/logout", sv.RequireAuth(sv.HandleLogout))
	mux.HandleFunc("/api/history", sv.RequireAuth(sv.HandleHistory))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				server.WriteAPIError(w, http.StatusInternalServerError, nil, "internal server error")
			}
			log.Printf("[%s] %s %s\n", time.Now().Format(time.RFC3339),
                      r.Method, r.URL.Path)
		}()
		mux.ServeHTTP(w, r)
	})

	log.Infof("Listening on %s", sv.Addr)
	log.Abortf("%s", http.ListenAndServe(sv.Addr, handler))
}
