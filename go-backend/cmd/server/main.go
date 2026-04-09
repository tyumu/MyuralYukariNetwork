package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go-backend/internal/api"
	"go-backend/internal/config"
	"go-backend/internal/logger"
	"go-backend/internal/memory"
	llm "go-backend/internal/ollama"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}
	fmt.Print(cfg.String())

	// Initialize logger
	logLevel := logger.ParseLevelString(cfg.Log.Level)
	log := logger.NewLogger(logLevel, cfg.Log.DevMode)

	// Initialize clients
	memClient := memory.NewClient(cfg.Memory.Endpoint, log)
	llmClient := llm.NewClient(cfg.LLM.BaseURL, log)

	// Initialize API handler
	handler := api.NewHandler(memClient, llmClient, cfg.LLM.Model, cfg.Chat.RecallTopK, log)
	statusCache := newStatusCache(30 * time.Second)

	// Setup routes
	http.HandleFunc("/chat", handler.Chat)
	http.HandleFunc("/status", statusCache.wrap(handler.Status, log))

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info("server starting", "addr", addr)

	go func() {
		if err := http.ListenAndServe(addr, loggingMiddleware(log, cfg.Frontend.Origin)); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("server shutting down")
}

func loggingMiddleware(log *logger.Logger, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions && r.URL.Path != "/status" && r.URL.Path != "/health" {
			log.Debug("http request", "method", r.Method, "path", r.RequestURI)
		}

		// Allow the Tauri dev server to call the Go API directly.
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		http.DefaultServeMux.ServeHTTP(w, r)
	})
}

type statusCache struct {
	mu        sync.Mutex
	updatedAt time.Time
	cacheBody []byte
	cacheCode int
	ttl       time.Duration
}

func newStatusCache(ttl time.Duration) *statusCache {
	return &statusCache{ttl: ttl}
}

func (c *statusCache) wrap(next http.HandlerFunc, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next(w, r)
			return
		}

		c.mu.Lock()
		fresh := !c.updatedAt.IsZero() && time.Since(c.updatedAt) < c.ttl && len(c.cacheBody) > 0
		body := append([]byte(nil), c.cacheBody...)
		code := c.cacheCode
		c.mu.Unlock()

		if fresh {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(code)
			_, _ = w.Write(body)
			return
		}

		rw := &responseCapture{ResponseWriter: w}
		next(rw, r)

		c.mu.Lock()
		c.updatedAt = time.Now()
		c.cacheCode = rw.statusCode
		c.cacheBody = append(c.cacheBody[:0], rw.body.Bytes()...)
		c.mu.Unlock()
		log.Debug("status cache refreshed")
	}
}

type responseCapture struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (r *responseCapture) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCapture) Write(p []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	_, _ = r.body.Write(p)
	return r.ResponseWriter.Write(p)
}
