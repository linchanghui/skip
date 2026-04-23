package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"skip/internal/db"
	"skip/internal/httpserver"
	"skip/internal/repository"
)

// rewriteRedirectLocation rewrites Location headers so subpath deployments don't redirect to "/".
func rewriteRedirectLocation(baseNoSlash string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(&redirectPrefixRW{ResponseWriter: w, base: baseNoSlash}, r)
	})
}

type redirectPrefixRW struct {
	http.ResponseWriter
	base string
	done bool
}

func (w *redirectPrefixRW) WriteHeader(code int) {
	if !w.done {
		w.done = true
		loc := w.Header().Get("Location")
		if loc != "" && strings.HasPrefix(loc, "/") && !strings.HasPrefix(loc, w.base+"/") && loc != w.base {
			w.Header().Set("Location", w.base+loc)
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *redirectPrefixRW) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", filepath.Join("data", "app.db"), "sqlite database file path")
	basePath := flag.String("base", "", "Mount base path, for example /skip (must match reverse proxy path and VITE_BASE_PATH).")
	staticPath := flag.String("static", "", "SPA static directory, for example web/dist.")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		log.Error("mkdir data dir", "err", err)
		os.Exit(1)
	}

	dsn := "file:" + *dbPath + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Error("open sqlite", "err", err)
		os.Exit(1)
	}
	sqlDB.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.Migrate(ctx, func(ctx context.Context, query string) error {
		_, err := sqlDB.ExecContext(ctx, query)
		return err
	}); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	repo := &repository.Store{DB: sqlDB}
	if err := repo.SeedDemo(ctx); err != nil {
		log.Error("seed", "err", err)
		os.Exit(1)
	}

	adminKey := strings.TrimSpace(os.Getenv("SKIP_ADMIN_API_KEY"))
	if adminKey == "" {
		log.Warn("SKIP_ADMIN_API_KEY not set; POST /v1/stores/{id}/status-reports will return 503")
	}

	srv := &httpserver.Server{
		Log:       log,
		Repo:      repo,
		AdminKey:  adminKey,
		StaticDir: strings.TrimSpace(*staticPath),
	}

	var handler http.Handler = srv.Handler()

	bp := strings.TrimSpace(*basePath)
	if bp != "" {
		if !strings.HasPrefix(bp, "/") {
			log.Error("-base must start with /, e.g. /skip")
			os.Exit(1)
		}
		bp = strings.TrimSuffix(bp, "/")
		prefix := bp + "/"
		strip := rewriteRedirectLocation(bp, http.StripPrefix(bp, handler))
		root := http.NewServeMux()
		root.HandleFunc(bp, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == bp {
				http.Redirect(w, r, prefix, http.StatusFound)
				return
			}
			http.NotFound(w, r)
		})
		root.Handle(prefix, strip)
		handler = root
		log.Info("app mounted under prefix", "path", prefix)
	}

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", *addr, "db", *dbPath, "base", strings.TrimSpace(*basePath), "static", strings.TrimSpace(*staticPath))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http", "err", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	_ = sqlDB.Close()
}
