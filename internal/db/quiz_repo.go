package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/models"
)

var (
	// ErrForbidden indicates the user is not authorized to access or modify the entity.
	ErrForbidden = errors.New("forbidden")
)

// QuizRepo provides database operations for quizzes, questions, and options.
type QuizRepo struct {
	db *sql.DB
}

// NewQuizRepo creates a new QuizRepo.
func NewQuizRepo(db *sql.DB) *QuizRepo {
	return &QuizRepo{db: db}
}

// CreateQuiz inserts a quiz with its questions and options inside an atomic transaction.
func (r *QuizRepo) CreateQuiz(ctx context.Context, quiz *models.Quiz) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if quiz.ID == "" {
		quiz.ID = uuid.NewString()
	}

	quizQuery := `
		INSERT INTO quizzes (id, instructor_id, title, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = tx.ExecContext(ctx, quizQuery, quiz.ID, quiz.InstructorID, quiz.Title, quiz.Description, quiz.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert quiz: %w", err)
	}

	questionQuery := `
		INSERT INTO questions (id, quiz_id, question_text, question_type, time_limit_sec, points, order_index)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	optionQuery := `
		INSERT INTO options (id, question_id, option_text, is_correct, order_index)
		VALUES (?, ?, ?, ?, ?)
	`

	for qIdx, question := range quiz.Questions {
		if question.ID == "" {
			question.ID = uuid.NewString()
		}
		question.QuizID = quiz.ID
		question.OrderIndex = qIdx

		if question.QuestionType == "" {
			question.QuestionType = "MULTIPLE_CHOICE"
		}
		if question.TimeLimitSec <= 0 {
			question.TimeLimitSec = 30
		}
		if question.Points <= 0 {
			question.Points = 1000
		}

		_, err = tx.ExecContext(ctx, questionQuery,
			question.ID,
			question.QuizID,
			question.QuestionText,
			question.QuestionType,
			question.TimeLimitSec,
			question.Points,
			question.OrderIndex,
		)
		if err != nil {
			return fmt.Errorf("failed to insert question %d: %w", qIdx, err)
		}

		for oIdx, option := range question.Options {
			if option.ID == "" {
				option.ID = uuid.NewString()
			}
			option.QuestionID = question.ID
			option.OrderIndex = oIdx

			_, err = tx.ExecContext(ctx, optionQuery,
				option.ID,
				option.QuestionID,
				option.OptionText,
				option.IsCorrect,
				option.OrderIndex,
			)
			if err != nil {
				return fmt.Errorf("failed to insert option %d for question %d: %w", oIdx, qIdx, err)
			}
			question.Options[oIdx] = option
		}
		quiz.Questions[qIdx] = question
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit quiz transaction: %w", err)
	}

	quiz.QuestionCount = len(quiz.Questions)
	return nil
}

// GetQuizzesByInstructorID retrieves all quizzes owned by an instructor.
func (r *QuizRepo) GetQuizzesByInstructorID(ctx context.Context, instructorID string) ([]models.Quiz, error) {
	query := `
		SELECT q.id, q.instructor_id, q.title, COALESCE(q.description, ''), q.created_at, COUNT(ques.id) as question_count
		FROM quizzes q
		LEFT JOIN questions ques ON q.id = ques.quiz_id
		WHERE q.instructor_id = ?
		GROUP BY q.id
		ORDER BY q.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, instructorID)
	if err != nil {
		return nil, fmt.Errorf("failed to query quizzes: %w", err)
	}
	defer rows.Close()

	quizzes := make([]models.Quiz, 0)
	for rows.Next() {
		var q models.Quiz
		if err := rows.Scan(&q.ID, &q.InstructorID, &q.Title, &q.Description, &q.CreatedAt, &q.QuestionCount); err != nil {
			return nil, fmt.Errorf("failed to scan quiz: %w", err)
		}
		quizzes = append(quizzes, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during quiz rows iteration: %w", err)
	}
	return quizzes, nil
}

// GetQuizByID retrieves a full quiz by ID, including its ordered questions and options.
func (r *QuizRepo) GetQuizByID(ctx context.Context, id string) (*models.Quiz, error) {
	quizQuery := `
		SELECT id, instructor_id, title, COALESCE(description, ''), created_at
		FROM quizzes
		WHERE id = ?
	`
	var quiz models.Quiz
	err := r.db.QueryRowContext(ctx, quizQuery, id).Scan(
		&quiz.ID,
		&quiz.InstructorID,
		&quiz.Title,
		&quiz.Description,
		&quiz.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query quiz: %w", err)
	}

	// Fetch questions ordered by order_index
	questionQuery := `
		SELECT id, quiz_id, question_text, question_type, time_limit_sec, points, order_index
		FROM questions
		WHERE quiz_id = ?
		ORDER BY order_index ASC
	`
	qRows, err := r.db.QueryContext(ctx, questionQuery, quiz.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query questions: %w", err)
	}
	defer qRows.Close()

	questionMap := make(map[string]*models.Question)
	questions := make([]models.Question, 0)

	for qRows.Next() {
		var q models.Question
		if err := qRows.Scan(
			&q.ID,
			&q.QuizID,
			&q.QuestionText,
			&q.QuestionType,
			&q.TimeLimitSec,
			&q.Points,
			&q.OrderIndex,
		); err != nil {
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}
		q.Options = make([]models.Option, 0)
		questions = append(questions, q)
	}
	if err := qRows.Err(); err != nil {
		return nil, fmt.Errorf("error during question rows iteration: %w", err)
	}

	// Link pointers in order
	for i := range questions {
		questionMap[questions[i].ID] = &questions[i]
	}

	// Fetch options for all questions in this quiz
	optionQuery := `
		SELECT o.id, o.question_id, o.option_text, o.is_correct, o.order_index
		FROM options o
		JOIN questions q ON o.question_id = q.id
		WHERE q.quiz_id = ?
		ORDER BY o.question_id, o.order_index ASC
	`
	oRows, err := r.db.QueryContext(ctx, optionQuery, quiz.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query options: %w", err)
	}
	defer oRows.Close()

	for oRows.Next() {
		var opt models.Option
		if err := oRows.Scan(
			&opt.ID,
			&opt.QuestionID,
			&opt.OptionText,
			&opt.IsCorrect,
			&opt.OrderIndex,
		); err != nil {
			return nil, fmt.Errorf("failed to scan option: %w", err)
		}
		if qPtr, exists := questionMap[opt.QuestionID]; exists {
			qPtr.Options = append(qPtr.Options, opt)
		}
	}
	if err := oRows.Err(); err != nil {
		return nil, fmt.Errorf("error during option rows iteration: %w", err)
	}

	quiz.Questions = questions
	quiz.QuestionCount = len(questions)
	return &quiz, nil
}

// DeleteQuiz deletes a quiz verifying that it belongs to the given instructorID.
func (r *QuizRepo) DeleteQuiz(ctx context.Context, id string, instructorID string) error {
	var ownerID string
	err := r.db.QueryRowContext(ctx, "SELECT instructor_id FROM quizzes WHERE id = ?", id).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to find quiz: %w", err)
	}

	if ownerID != instructorID {
		return ErrForbidden
	}

	res, err := r.db.ExecContext(ctx, "DELETE FROM quizzes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete quiz: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
