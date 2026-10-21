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

// QuizHandler handles REST operations for quizzes.
type QuizHandler struct {
	quizRepo *db.QuizRepo
}

// NewQuizHandler creates a new QuizHandler.
func NewQuizHandler(quizRepo *db.QuizRepo) *QuizHandler {
	return &QuizHandler{quizRepo: quizRepo}
}

// CreateQuiz creates a new quiz with nested questions and options.
func (h *QuizHandler) CreateQuiz(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.CreateQuizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid json request body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		Error(w, http.StatusBadRequest, "quiz title is required")
		return
	}
	if len(req.Questions) == 0 {
		Error(w, http.StatusBadRequest, "at least one question is required")
		return
	}

	questions := make([]models.Question, len(req.Questions))
	for qIdx, qReq := range req.Questions {
		qText := strings.TrimSpace(qReq.QuestionText)
		if qText == "" {
			Error(w, http.StatusBadRequest, "question text cannot be empty")
			return
		}
		if len(qReq.Options) < 2 {
			Error(w, http.StatusBadRequest, "each question must have at least 2 options")
			return
		}

		timeLimit := qReq.TimeLimitSec
		if timeLimit <= 0 {
			timeLimit = 30
		}
		points := qReq.Points
		if points <= 0 {
			points = 1000
		}
		qType := strings.TrimSpace(qReq.QuestionType)
		if qType == "" {
			qType = "MULTIPLE_CHOICE"
		}

		options := make([]models.Option, len(qReq.Options))
		for oIdx, oReq := range qReq.Options {
			optText := strings.TrimSpace(oReq.OptionText)
			if optText == "" {
				Error(w, http.StatusBadRequest, "option text cannot be empty")
				return
			}
			options[oIdx] = models.Option{
				ID:         uuid.NewString(),
				OptionText: optText,
				IsCorrect:  oReq.IsCorrect,
				OrderIndex: oIdx,
			}
		}

		questions[qIdx] = models.Question{
			ID:           uuid.NewString(),
			QuestionText: qText,
			QuestionType: qType,
			TimeLimitSec: timeLimit,
			Points:       points,
			OrderIndex:   qIdx,
			Options:      options,
		}
	}

	quiz := models.Quiz{
		ID:            uuid.NewString(),
		InstructorID:  claims.UserID,
		Title:         req.Title,
		Description:   strings.TrimSpace(req.Description),
		QuestionCount: len(questions),
		CreatedAt:     time.Now().UTC(),
		Questions:     questions,
	}

	if err := h.quizRepo.CreateQuiz(r.Context(), &quiz); err != nil {
		Error(w, http.StatusInternalServerError, "failed to create quiz")
		return
	}

	JSON(w, http.StatusCreated, quiz)
}

// ListQuizzes lists all quizzes belonging to the authenticated instructor.
func (h *QuizHandler) ListQuizzes(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	quizzes, err := h.quizRepo.GetQuizzesByInstructorID(r.Context(), claims.UserID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to retrieve quizzes")
		return
	}

	JSON(w, http.StatusOK, quizzes)
}

// GetQuiz retrieves a single quiz with all nested questions and options.
func (h *QuizHandler) GetQuiz(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	quizID := r.PathValue("id")
	if quizID == "" {
		Error(w, http.StatusBadRequest, "missing quiz id")
		return
	}

	quiz, err := h.quizRepo.GetQuizByID(r.Context(), quizID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			Error(w, http.StatusNotFound, "quiz not found")
			return
		}
		Error(w, http.StatusInternalServerError, "failed to retrieve quiz")
		return
	}

	if quiz.InstructorID != claims.UserID {
		Error(w, http.StatusForbidden, "forbidden: access denied")
		return
	}

	JSON(w, http.StatusOK, quiz)
}

// DeleteQuiz deletes a quiz owned by the authenticated instructor.
func (h *QuizHandler) DeleteQuiz(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.UserFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	quizID := r.PathValue("id")
	if quizID == "" {
		Error(w, http.StatusBadRequest, "missing quiz id")
		return
	}

	err := h.quizRepo.DeleteQuiz(r.Context(), quizID, claims.UserID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			Error(w, http.StatusNotFound, "quiz not found")
			return
		}
		if errors.Is(err, db.ErrForbidden) {
			Error(w, http.StatusForbidden, "forbidden: access denied")
			return
		}
		Error(w, http.StatusInternalServerError, "failed to delete quiz")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "quiz deleted"})
}
