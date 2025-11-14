package commands

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserCommandHandler handles all write operations for users
type UserCommandHandler struct {
	db *sql.DB
}

// NewUserCommandHandler creates a new command handler
func NewUserCommandHandler(db *sql.DB) *UserCommandHandler {
	return &UserCommandHandler{db: db}
}

// RegisterUser processes RegisterUserCommand
func (h *UserCommandHandler) RegisterUser(cmd RegisterUserCommand) (*CommandResult, error) {
	// Validation
	if err := h.validateRegister(cmd); err != nil {
		return &CommandResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Check if email/username already exists
	var exists bool
	err := h.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = ? OR username = ?)",
		cmd.Email, cmd.Username,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return &CommandResult{
			Success: false,
			Error:   "email or username already exists",
		}, nil
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert user
	result, err := h.db.Exec(
		"INSERT INTO users (email, username, password) VALUES (?, ?, ?)",
		cmd.Email, cmd.Username, string(hashedPassword),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"user_id":  userID,
			"username": cmd.Username,
		},
	}, nil
}

// Login processes LoginCommand
func (h *UserCommandHandler) Login(cmd LoginCommand) (*CommandResult, error) {
	// Validation
	if err := h.validateLogin(cmd); err != nil {
		return &CommandResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Find user by email or username
	var userID int
	var email, username, password string
	err := h.db.QueryRow(
		"SELECT id, email, username, password FROM users WHERE email = ? OR username = ?",
		cmd.EmailOrUsername, cmd.EmailOrUsername,
	).Scan(&userID, &email, &username, &password)

	if err != nil {
		if err == sql.ErrNoRows {
			return &CommandResult{
				Success: false,
				Error:   "invalid credentials",
			}, nil
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(password), []byte(cmd.Password))
	if err != nil {
		return &CommandResult{
			Success: false,
			Error:   "invalid credentials",
		}, nil
	}

	// Create session
	sessionID, err := h.createSession(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"user_id":    userID,
			"username":   username,
			"session_id": sessionID,
		},
	}, nil
}

// Logout removes user session
func (h *UserCommandHandler) Logout(userID int) (*CommandResult, error) {
	_, err := h.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete session: %w", err)
	}

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"message": "logged out successfully",
		},
	}, nil
}

// createSession generates a new session for the user
func (h *UserCommandHandler) createSession(userID int) (string, error) {
	sessionID := generateSessionID()
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hour session

	// Delete old session if exists
	_, err := h.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return "", fmt.Errorf("failed to delete old session: %w", err)
	}

	// Insert new session
	_, err = h.db.Exec(
		"INSERT INTO sessions (user_id, session_id, expires_at) VALUES (?, ?, ?)",
		userID, sessionID, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert session: %w", err)
	}

	return sessionID, nil
}

// Validation methods

func (h *UserCommandHandler) validateRegister(cmd RegisterUserCommand) error {
	email := strings.TrimSpace(cmd.Email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return fmt.Errorf("invalid email format")
	}

	username := strings.TrimSpace(cmd.Username)
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if len(username) > 50 {
		return fmt.Errorf("username must be less than 50 characters")
	}

	if cmd.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(cmd.Password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	return nil
}

func (h *UserCommandHandler) validateLogin(cmd LoginCommand) error {
	if strings.TrimSpace(cmd.EmailOrUsername) == "" {
		return fmt.Errorf("email or username is required")
	}
	if cmd.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	// Simple session ID generation (in production, use crypto/rand)
	return fmt.Sprintf("session_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}
