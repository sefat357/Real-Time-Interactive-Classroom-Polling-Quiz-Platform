package api

import (
	"net/http"

	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/auth"
)

// RouterConfig contains dependencies required to build the HTTP routes.
type RouterConfig struct {
	AuthHandler  *AuthHandler
	QuizHandler  *QuizHandler
	TokenService *auth.TokenService
}

// NewRouter constructs the http.Handler for the application using Go 1.22+ ServeMux.
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Public Auth endpoints
	mux.HandleFunc("POST /api/auth/register", cfg.AuthHandler.Register)
	mux.HandleFunc("POST /api/auth/login", cfg.AuthHandler.Login)

	// Protected Auth endpoints
	protectedMe := cfg.TokenService.Middleware(http.HandlerFunc(cfg.AuthHandler.Me))
	mux.Handle("GET /api/auth/me", protectedMe)

	// Protected Quiz endpoints
	createQuiz := cfg.TokenService.Middleware(http.HandlerFunc(cfg.QuizHandler.CreateQuiz))
	listQuizzes := cfg.TokenService.Middleware(http.HandlerFunc(cfg.QuizHandler.ListQuizzes))
	getQuiz := cfg.TokenService.Middleware(http.HandlerFunc(cfg.QuizHandler.GetQuiz))
	deleteQuiz := cfg.TokenService.Middleware(http.HandlerFunc(cfg.QuizHandler.DeleteQuiz))

	mux.Handle("POST /api/quizzes", createQuiz)
	mux.Handle("GET /api/quizzes", listQuizzes)
	mux.Handle("GET /api/quizzes/{id}", getQuiz)
	mux.Handle("DELETE /api/quizzes/{id}", deleteQuiz)

	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
