package models

import "time"

// User represents an instructor account.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserResponse is the safe public representation of a User.
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRequest contains registration payload.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest contains login credentials.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse returns the JWT token and user profile.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// Option represents a choice for a quiz question.
type Option struct {
	ID         string `json:"id"`
	QuestionID string `json:"question_id"`
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct"`
	OrderIndex int    `json:"order_index"`
}

// Question represents a quiz question with its choices.
type Question struct {
	ID           string   `json:"id"`
	QuizID       string   `json:"quiz_id"`
	QuestionText string   `json:"question_text"`
	QuestionType string   `json:"question_type"`
	TimeLimitSec int      `json:"time_limit_sec"`
	Points       int      `json:"points"`
	OrderIndex   int      `json:"order_index"`
	Options      []Option `json:"options"`
}

// Quiz represents a quiz and its questions.
type Quiz struct {
	ID            string     `json:"id"`
	InstructorID  string     `json:"instructor_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	QuestionCount int        `json:"question_count,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	Questions     []Question `json:"questions,omitempty"`
}

// CreateOptionRequest represents an option in a quiz creation payload.
type CreateOptionRequest struct {
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct"`
}

// CreateQuestionRequest represents a question in a quiz creation payload.
type CreateQuestionRequest struct {
	QuestionText string                `json:"question_text"`
	QuestionType string                `json:"question_type,omitempty"`
	TimeLimitSec int                   `json:"time_limit_sec,omitempty"`
	Points       int                   `json:"points,omitempty"`
	Options      []CreateOptionRequest `json:"options"`
}

// CreateQuizRequest represents the full payload for creating a quiz.
type CreateQuizRequest struct {
	Title       string                  `json:"title"`
	Description string                  `json:"description,omitempty"`
	Questions   []CreateQuestionRequest `json:"questions"`
}

// Session represents a live quiz room run by an instructor.
type Session struct {
	ID           string     `json:"id"`
	QuizID       string     `json:"quiz_id"`
	InstructorID string     `json:"instructor_id"`
	PIN          string     `json:"pin"`
	Status       string     `json:"status"` // LOBBY, QUESTION_ACTIVE, QUESTION_REVEAL, COMPLETED
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
}

// Response represents a student's answer submission for a question.
type Response struct {
	ID               string    `json:"id"`
	SessionID        string    `json:"session_id"`
	QuestionID       string    `json:"question_id"`
	StudentID        string    `json:"student_id"`
	StudentNickname  string    `json:"student_nickname"`
	SelectedOptionID *string   `json:"selected_option_id,omitempty"`
	IsCorrect        bool      `json:"is_correct"`
	ResponseTimeMS   int       `json:"response_time_ms"`
	ScoreAwarded     int       `json:"score_awarded"`
	SubmittedAt      time.Time `json:"submitted_at"`
}

// QAQuestion represents a live Q&A question posted by a student during a session.
type QAQuestion struct {
	ID              string    `json:"id"`
	SessionID       string    `json:"session_id"`
	StudentID       string    `json:"student_id"`
	StudentNickname string    `json:"student_nickname"`
	Content         string    `json:"content"`
	Upvotes         int       `json:"upvotes"`
	IsAnswered      bool      `json:"is_answered"`
	CreatedAt       time.Time `json:"created_at"`
}
