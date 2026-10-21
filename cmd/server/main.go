package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/api"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/auth"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/config"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/db"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("[INFO] Starting server on port %s...", cfg.Port)

	// Initialize SQLite with WAL mode & run migrations
	database, err := db.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize database: %v", err)
	}
	defer database.Close()
	log.Printf("[INFO] Database initialized at %s with WAL mode", cfg.DBPath)

	// Repositories & Services
	userRepo := db.NewUserRepo(database)
	quizRepo := db.NewQuizRepo(database)
	tokenService := auth.NewTokenService(cfg.JWTSecret)

	// Handlers
	authHandler := api.NewAuthHandler(userRepo, tokenService)
	quizHandler := api.NewQuizHandler(quizRepo)

	// Router
	router := api.NewRouter(api.RouterConfig{
		AuthHandler:  authHandler,
		QuizHandler:  quizHandler,
		TokenService: tokenService,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background goroutine
	go func() {
		log.Printf("[INFO] HTTP server listening on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[INFO] Received signal '%s', initiating graceful shutdown...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] Server shutdown error: %v", err)
	}

	log.Println("[INFO] Server exited cleanly.")
}
