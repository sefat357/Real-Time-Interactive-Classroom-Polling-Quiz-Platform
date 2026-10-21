package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/auth"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/db"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/models"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	userRepo     *db.UserRepo
	tokenService *auth.TokenService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(userRepo *db.UserRepo, tokenService *auth.TokenService) *AuthHandler {
	return &AuthHandler{
		userRepo:     userRepo,
		tokenService: tokenService,
	}
}

// Register registers a new instructor account.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid json request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Name == "" {
		Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		Error(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if len(req.Password) < 6 {
		Error(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	user := models.User{
		ID:           uuid.NewString(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}

	if err := h.userRepo.CreateUser(r.Context(), &user); err != nil {
		if errors.Is(err, db.ErrDuplicateEmail) {
			Error(w, http.StatusConflict, "email already registered")
			return
		}
		Error(w, http.StatusInternalServerError, "failed to create user account")
		return
	}

	token, err := h.tokenService.GenerateToken(&user)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to generate authentication token")
		return
	}

	res := models.AuthResponse{
		Token: token,
		User: models.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	}
	JSON(w, http.StatusCreated, res)
}

// Login authenticates an instructor and returns a JWT.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid json request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.userRepo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			Error(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		Error(w, http.StatusInternalServerError, "database error")
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := h.tokenService.GenerateToken(user)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to generate authentication token")
		return
	}

	res := models.AuthResponse{
		Token: token,
		User: models.UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	}
	JSON(w, http.StatusOK, res)
}

// Me returns the profile of the currently authenticated instructor.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.userRepo.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			Error(w, http.StatusNotFound, "user not found")
			return
		}
		Error(w, http.StatusInternalServerError, "database error")
		return
	}

	res := models.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
	JSON(w, http.StatusOK, res)
}
