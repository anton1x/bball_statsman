package main

import (
	"context"
	"log"
	nethttp "net/http"
	"os"
	"strings"
	"time"

	"bball_statsman_backend/internal/infrastructure/filedb"
	transport "bball_statsman_backend/internal/interface/http"
	"bball_statsman_backend/internal/usecase"
)

func main() {
	dbPath := getenv("DB_PATH", "./data/statsman.json")
	listenAddr := getenv("LISTEN_ADDR", ":8080")

	repo := filedb.NewVideoStateRepository(dbPath)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repo.InitSchema(ctx); err != nil {
		log.Fatalf("failed to init schema: %v", err)
	}

	uc := usecase.NewVideoStateUseCase(repo)
	h := transport.NewHandler(uc)

	mux := nethttp.NewServeMux()
	h.Register(mux)

	server := &nethttp.Server{
		Addr:    listenAddr,
		Handler: withCORS(mux, getenv("CORS_ALLOW_ORIGINS", "https://tripledouble.ru,http://localhost:5173,http://127.0.0.1:5173")),
	}

	log.Printf("backend listening on %s", listenAddr)
	if err := server.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func withCORS(next nethttp.Handler, allowedOriginsCSV string) nethttp.Handler {
	allowedOrigins := map[string]struct{}{}
	for _, origin := range strings.Split(allowedOriginsCSV, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		allowedOrigins[trimmed] = struct{}{}
	}

	isAllowedOrigin := func(origin string) bool {
		if origin == "" {
			return false
		}

		if _, ok := allowedOrigins[origin]; ok {
			return true
		}

		return strings.HasPrefix(origin, "https://") && strings.HasSuffix(origin, ".github.io")
	}

	return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		origin := r.Header.Get("Origin")
		if isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == nethttp.MethodOptions {
			if !isAllowedOrigin(origin) {
				w.WriteHeader(nethttp.StatusForbidden)
				return
			}

			w.WriteHeader(nethttp.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
