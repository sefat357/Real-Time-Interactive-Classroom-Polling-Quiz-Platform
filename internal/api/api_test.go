package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/api"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/auth"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/db"
	"github.com/sefat357/Real-Time-Interactive-Classroom-Polling-Quiz-Platform/internal/models"
)

func setupTestApp(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init test database: %v", err)
	}

	userRepo := db.NewUserRepo(database)
	quizRepo := db.NewQuizRepo(database)
	tokenService := auth.NewTokenService("test-secret-key-123456789")

	authHandler := api.NewAuthHandler(userRepo, tokenService)
	quizHandler := api.NewQuizHandler(quizRepo)

	router := api.NewRouter(api.RouterConfig{
		AuthHandler:  authHandler,
		QuizHandler:  quizHandler,
		TokenService: tokenService,
	})

	return router, database
}

func TestHealthEndpoints(t *testing.T) {
	router, database := setupTestApp(t)
	defer database.Close()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /health, got %d", rec.Code)
	}

	apiReq := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	apiRec := httptest.NewRecorder()
	router.ServeHTTP(apiRec, apiReq)

	if apiRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/health, got %d", apiRec.Code)
	}
}

func TestAuthFlow(t *testing.T) {
	router, database := setupTestApp(t)
	defer database.Close()

	// 1. Successful Registration
	regPayload := models.RegisterRequest{
		Name:     "Prof. Alan Turing",
		Email:    "alan@sust.edu",
		Password: "securepassword123",
	}
	body, _ := json.Marshal(regPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on register, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var regRes models.AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&regRes); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}
	if regRes.Token == "" {
		t.Fatalf("expected non-empty JWT token")
	}
	if regRes.User.Email != "alan@sust.edu" {
		t.Fatalf("expected email alan@sust.edu, got %s", regRes.User.Email)
	}

	// 2. Duplicate Registration Rejection (409 Conflict)
	dupReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	dupReq.Header.Set("Content-Type", "application/json")
	dupRec := httptest.NewRecorder()
	router.ServeHTTP(dupRec, dupReq)

	if dupRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on duplicate email, got %d", dupRec.Code)
	}

	// 3. Validation Edge Cases
	invalidCases := []struct {
		name     string
		payload  models.RegisterRequest
		expected int
	}{
		{"empty name", models.RegisterRequest{Name: "", Email: "test@example.com", Password: "password123"}, http.StatusBadRequest},
		{"invalid email", models.RegisterRequest{Name: "Name", Email: "invalidemail", Password: "password123"}, http.StatusBadRequest},
		{"short password", models.RegisterRequest{Name: "Name", Email: "short@example.com", Password: "123"}, http.StatusBadRequest},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.payload)
			r := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(b))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != tc.expected {
				t.Fatalf("expected status %d, got %d", tc.expected, w.Code)
			}
		})
	}

	// 4. Successful Login
	loginPayload := models.LoginRequest{
		Email:    "alan@sust.edu",
		Password: "securepassword123",
	}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login, got %d. Body: %s", loginRec.Code, loginRec.Body.String())
	}

	var loginRes models.AuthResponse
	if err := json.NewDecoder(loginRec.Body).Decode(&loginRes); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	if loginRes.Token == "" {
		t.Fatalf("expected token in login response")
	}

	// 5. Invalid Password Rejection (401)
	badPwPayload := models.LoginRequest{
		Email:    "alan@sust.edu",
		Password: "wrongpassword",
	}
	badPwBody, _ := json.Marshal(badPwPayload)
	badPwReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(badPwBody))
	badPwReq.Header.Set("Content-Type", "application/json")
	badPwRec := httptest.NewRecorder()
	router.ServeHTTP(badPwRec, badPwReq)

	if badPwRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized on wrong password, got %d", badPwRec.Code)
	}

	// 6. GET /api/auth/me (Protected)
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/auth/me, got %d", meRec.Code)
	}
	var meRes models.UserResponse
	if err := json.NewDecoder(meRec.Body).Decode(&meRes); err != nil {
		t.Fatalf("failed to decode me response: %v", err)
	}
	if meRes.Email != "alan@sust.edu" {
		t.Fatalf("expected email alan@sust.edu, got %s", meRes.Email)
	}

	// 7. GET /api/auth/me with invalid token
	badTokenReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	badTokenReq.Header.Set("Authorization", "Bearer invalid-token-string")
	badTokenRec := httptest.NewRecorder()
	router.ServeHTTP(badTokenRec, badTokenReq)

	if badTokenRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized on invalid token, got %d", badTokenRec.Code)
	}
}

func TestQuizCRUDAndIsolation(t *testing.T) {
	router, database := setupTestApp(t)
	defer database.Close()

	// Helper to register an instructor and obtain their token
	registerInstructor := func(name, email, password string) string {
		payload := models.RegisterRequest{Name: name, Email: email, Password: password}
		b, _ := json.Marshal(payload)
		r := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		var res models.AuthResponse
		_ = json.NewDecoder(w.Body).Decode(&res)
		return res.Token
	}

	tokenA := registerInstructor("Instructor A", "inst_a@sust.edu", "password123")
	tokenB := registerInstructor("Instructor B", "inst_b@sust.edu", "password123")

	// 1. Unauthorized Access Rejection on /api/quizzes
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/quizzes", nil)
	unauthRec := httptest.NewRecorder()
	router.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated GET /api/quizzes, got %d", unauthRec.Code)
	}

	// 2. Create Quiz Validation Errors
	invalidQuizCases := []struct {
		name     string
		payload  models.CreateQuizRequest
		expected int
	}{
		{"empty title", models.CreateQuizRequest{Title: "", Questions: []models.CreateQuestionRequest{
			{QuestionText: "Q?", Options: []models.CreateOptionRequest{{OptionText: "A", IsCorrect: true}, {OptionText: "B", IsCorrect: false}}},
		}}, http.StatusBadRequest},
		{"no questions", models.CreateQuizRequest{Title: "Title", Questions: []models.CreateQuestionRequest{}}, http.StatusBadRequest},
		{"question with 1 option", models.CreateQuizRequest{Title: "Title", Questions: []models.CreateQuestionRequest{
			{QuestionText: "Q?", Options: []models.CreateOptionRequest{{OptionText: "A", IsCorrect: true}}},
		}}, http.StatusBadRequest},
	}
	for _, tc := range invalidQuizCases {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.payload)
			r := httptest.NewRequest(http.MethodPost, "/api/quizzes", bytes.NewReader(b))
			r.Header.Set("Authorization", "Bearer "+tokenA)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != tc.expected {
				t.Fatalf("expected %d, got %d. Body: %s", tc.expected, w.Code, w.Body.String())
			}
		})
	}

	// 3. Successful Multi-Question Quiz Creation
	createPayload := models.CreateQuizRequest{
		Title:       "Operating Systems & Concurrency",
		Description: "Week 4 Live Quiz",
		Questions: []models.CreateQuestionRequest{
			{
				QuestionText: "Which Go primitive is best for fan-in state serialization?",
				QuestionType: "MULTIPLE_CHOICE",
				TimeLimitSec: 20,
				Points:       1000,
				Options: []models.CreateOptionRequest{
					{OptionText: "Channels", IsCorrect: true},
					{OptionText: "init() functions", IsCorrect: false},
					{OptionText: "defer statements", IsCorrect: false},
					{OptionText: "panic/recover", IsCorrect: false},
				},
			},
			{
				QuestionText: "What does SQLite WAL mode stand for?",
				QuestionType: "MULTIPLE_CHOICE",
				TimeLimitSec: 0, // Should default to 30
				Points:       0, // Should default to 1000
				Options: []models.CreateOptionRequest{
					{OptionText: "Write-Ahead Logging", IsCorrect: true},
					{OptionText: "Wide Access Layer", IsCorrect: false},
				},
			},
		},
	}

	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/api/quizzes", bytes.NewReader(createBody))
	createReq.Header.Set("Authorization", "Bearer "+tokenA)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on POST /api/quizzes, got %d. Body: %s", createRec.Code, createRec.Body.String())
	}

	var createdQuiz models.Quiz
	if err := json.NewDecoder(createRec.Body).Decode(&createdQuiz); err != nil {
		t.Fatalf("failed to decode created quiz: %v", err)
	}

	if createdQuiz.ID == "" {
		t.Fatalf("expected non-empty quiz id")
	}
	if len(createdQuiz.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(createdQuiz.Questions))
	}
	if createdQuiz.Questions[0].Options[0].OptionText != "Channels" {
		t.Fatalf("expected first option to be 'Channels', got %s", createdQuiz.Questions[0].Options[0].OptionText)
	}
	if createdQuiz.Questions[1].TimeLimitSec != 30 {
		t.Fatalf("expected default time_limit_sec 30, got %d", createdQuiz.Questions[1].TimeLimitSec)
	}
	if createdQuiz.Questions[1].Points != 1000 {
		t.Fatalf("expected default points 1000, got %d", createdQuiz.Questions[1].Points)
	}

	// 4. List Quizzes for Instructor A
	listReq := httptest.NewRequest(http.MethodGet, "/api/quizzes", nil)
	listReq.Header.Set("Authorization", "Bearer "+tokenA)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from GET /api/quizzes, got %d", listRec.Code)
	}

	var quizzesA []models.Quiz
	if err := json.NewDecoder(listRec.Body).Decode(&quizzesA); err != nil {
		t.Fatalf("failed to decode quizzes list: %v", err)
	}
	if len(quizzesA) != 1 {
		t.Fatalf("expected 1 quiz for instructor A, got %d", len(quizzesA))
	}
	if quizzesA[0].QuestionCount != 2 {
		t.Fatalf("expected question_count 2, got %d", quizzesA[0].QuestionCount)
	}

	// List Quizzes for Instructor B (should be empty)
	listReqB := httptest.NewRequest(http.MethodGet, "/api/quizzes", nil)
	listReqB.Header.Set("Authorization", "Bearer "+tokenB)
	listRecB := httptest.NewRecorder()
	router.ServeHTTP(listRecB, listReqB)
	var quizzesB []models.Quiz
	_ = json.NewDecoder(listRecB.Body).Decode(&quizzesB)
	if len(quizzesB) != 0 {
		t.Fatalf("expected 0 quizzes for instructor B, got %d", len(quizzesB))
	}

	// 5. Get Quiz by ID for Owner (Instructor A)
	getReq := httptest.NewRequest(http.MethodGet, "/api/quizzes/"+createdQuiz.ID, nil)
	getReq.Header.Set("Authorization", "Bearer "+tokenA)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from GET /api/quizzes/{id}, got %d. Body: %s", getRec.Code, getRec.Body.String())
	}

	var fetchedQuiz models.Quiz
	if err := json.NewDecoder(getRec.Body).Decode(&fetchedQuiz); err != nil {
		t.Fatalf("failed to decode fetched quiz: %v", err)
	}
	if len(fetchedQuiz.Questions) != 2 || len(fetchedQuiz.Questions[0].Options) != 4 {
		t.Fatalf("expected nested questions (2) and options (4), got %d and %d",
			len(fetchedQuiz.Questions), len(fetchedQuiz.Questions[0].Options))
	}

	// 6. Cross-Instructor Access Prevention (Instructor B accessing Instructor A's quiz)
	crossGetReq := httptest.NewRequest(http.MethodGet, "/api/quizzes/"+createdQuiz.ID, nil)
	crossGetReq.Header.Set("Authorization", "Bearer "+tokenB)
	crossGetRec := httptest.NewRecorder()
	router.ServeHTTP(crossGetRec, crossGetReq)

	if crossGetRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cross-instructor access, got %d", crossGetRec.Code)
	}

	// Cross-Instructor Delete Prevention
	crossDelReq := httptest.NewRequest(http.MethodDelete, "/api/quizzes/"+createdQuiz.ID, nil)
	crossDelReq.Header.Set("Authorization", "Bearer "+tokenB)
	crossDelRec := httptest.NewRecorder()
	router.ServeHTTP(crossDelRec, crossDelReq)

	if crossDelRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on cross-instructor delete, got %d", crossDelRec.Code)
	}

	// 7. Successful Deletion by Owner (Instructor A)
	delReq := httptest.NewRequest(http.MethodDelete, "/api/quizzes/"+createdQuiz.ID, nil)
	delReq.Header.Set("Authorization", "Bearer "+tokenA)
	delRec := httptest.NewRecorder()
	router.ServeHTTP(delRec, delReq)

	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on delete, got %d. Body: %s", delRec.Code, delRec.Body.String())
	}

	// Subsequent GET returns 404
	afterDelGet := httptest.NewRequest(http.MethodGet, "/api/quizzes/"+createdQuiz.ID, nil)
	afterDelGet.Header.Set("Authorization", "Bearer "+tokenA)
	afterDelRec := httptest.NewRecorder()
	router.ServeHTTP(afterDelRec, afterDelGet)

	if afterDelRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found after deletion, got %d", afterDelRec.Code)
	}

	// 8. Verify SQLite Database Cascade Deletion
	var questionCount, optionCount int
	_ = database.QueryRow("SELECT COUNT(*) FROM questions WHERE quiz_id = ?", createdQuiz.ID).Scan(&questionCount)
	if questionCount != 0 {
		t.Fatalf("expected 0 questions remaining after cascade delete, found %d", questionCount)
	}

	_ = database.QueryRow(`
		SELECT COUNT(*) FROM options o
		JOIN questions q ON o.question_id = q.id
		WHERE q.quiz_id = ?
	`, createdQuiz.ID).Scan(&optionCount)
	if optionCount != 0 {
		t.Fatalf("expected 0 options remaining after cascade delete, found %d", optionCount)
	}
}
