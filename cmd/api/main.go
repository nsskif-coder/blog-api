package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"blog-api/internal/auth"
	"blog-api/internal/handler"
	"blog-api/internal/logger"
	"blog-api/internal/middleware"
	"blog-api/internal/storage"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment")
	}

	auth.JWTSecret = os.Getenv("JWT_SECRET")
	if auth.JWTSecret == "" {
		auth.JWTSecret = "default-secret-key-change-in-production"
		log.Println("Warning: JWT_SECRET not set, using default value")
	}

	// папки для данных и логов
	os.MkdirAll("data", 0755)
	os.MkdirAll("logs", 0755)

	store, err := storage.NewStorage("data")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	appLogger := logger.NewLogger()
	appLogger.Start()

	h := handler.NewHandler(store, appLogger)

	// создаем маршрутизатор
	mux := http.NewServeMux()

	// эндпоинты
	mux.HandleFunc("/health", h.HealthCheck)
	mux.HandleFunc("/api/health", h.HealthCheck)

	mux.HandleFunc("/register", h.Register)
	mux.HandleFunc("/api/register", h.Register)

	mux.HandleFunc("/login", h.Login)
	mux.HandleFunc("/api/login", h.Login)

	mux.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetPosts(w, r)
		} else if r.Method == http.MethodPost {
			middleware.AuthMiddleware(h.CreatePost)(w, r)
		} else {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.GetPosts(w, r)
		} else if r.Method == http.MethodPost {
			middleware.AuthMiddleware(h.CreatePost)(w, r)
		} else {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/posts/", func(w http.ResponseWriter, r *http.Request) {
		routeHandler(w, r, h)
	})
	mux.HandleFunc("/api/posts/", func(w http.ResponseWriter, r *http.Request) {
		routeHandler(w, r, h)
	})

	// Конфигурация сервера
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf("%s:%s", host, port)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("Server starting on http://%s\n", addr)
		fmt.Printf("Health check: http://localhost:%s/health\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Ждём сигнала завершения
	<-quit
	fmt.Println("\n⏹  Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	appLogger.Stop()
	fmt.Println("✅ Server stopped gracefully")
}

func routeHandler(w http.ResponseWriter, r *http.Request, h *handler.Handler) {
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")


	if len(parts) > 0 && parts[0] == "api" {
		parts = parts[1:]
	}

	if len(parts) == 3 && parts[2] == "comments" {
		if r.Method == http.MethodGet {
			h.GetComments(w, r)
		} else if r.Method == http.MethodPost {
			middleware.AuthMiddleware(h.CreateComment)(w, r)
		} else {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 2 {
		if r.Method == http.MethodGet {
			h.GetPostByID(w, r)
		} else {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
}