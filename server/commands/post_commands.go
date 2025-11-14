package commands

import (
	"database/sql"
	"fmt"
	"strings"

	"forum/server/models"
)

// PostCommandHandler handles all write operations for posts
type PostCommandHandler struct {
	db *sql.DB
}

// NewPostCommandHandler creates a new command handler
func NewPostCommandHandler(db *sql.DB) *PostCommandHandler {
	return &PostCommandHandler{db: db}
}

// Handle processes CreatePostCommand
func (h *PostCommandHandler) CreatePost(cmd CreatePostCommand) (*CommandResult, error) {
	// Validation
	if err := h.validateCreatePost(cmd); err != nil {
		return &CommandResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Create post
	result, err := tx.Exec(
		"INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)",
		cmd.UserID, cmd.Title, cmd.Content,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert post: %w", err)
	}

	postID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get post ID: %w", err)
	}

	// Link categories
	for _, categoryID := range cmd.CategoryIDs {
		_, err := tx.Exec(
			"INSERT INTO post_category (post_id, category_id) VALUES (?, ?)",
			postID, categoryID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to link category %d: %w", categoryID, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"post_id": postID,
		},
	}, nil
}

// Handle processes CreateCommentCommand
func (h *PostCommandHandler) CreateComment(cmd CreateCommentCommand) (*CommandResult, error) {
	// Validation
	if err := h.validateCreateComment(cmd); err != nil {
		return &CommandResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Verify post exists
	var postExists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE id = ?)", cmd.PostID).Scan(&postExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check post existence: %w", err)
	}
	if !postExists {
		return &CommandResult{
			Success: false,
			Error:   "post not found",
		}, nil
	}

	// Insert comment
	result, err := h.db.Exec(
		"INSERT INTO comments (user_id, post_id, content) VALUES (?, ?, ?)",
		cmd.UserID, cmd.PostID, cmd.Content,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert comment: %w", err)
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get comment ID: %w", err)
	}

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"comment_id": commentID,
		},
	}, nil
}

// Handle processes ReactToPostCommand
func (h *PostCommandHandler) ReactToPost(cmd ReactToPostCommand) (*CommandResult, error) {
	// Validation
	if err := h.validateReaction(cmd.Reaction); err != nil {
		return &CommandResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Check if reaction already exists
	var existingReaction sql.NullString
	err := h.db.QueryRow(
		"SELECT reaction FROM post_reactions WHERE user_id = ? AND post_id = ?",
		cmd.UserID, cmd.PostID,
	).Scan(&existingReaction)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing reaction: %w", err)
	}

	// If same reaction, remove it (toggle off)
	if existingReaction.Valid && existingReaction.String == cmd.Reaction {
		_, err := h.db.Exec(
			"DELETE FROM post_reactions WHERE user_id = ? AND post_id = ?",
			cmd.UserID, cmd.PostID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to remove reaction: %w", err)
		}
		return &CommandResult{
			Success: true,
			Data: map[string]interface{}{
				"action": "removed",
			},
		}, nil
	}

	// Upsert reaction (insert or update)
	_, err = h.db.Exec(`
		INSERT INTO post_reactions (user_id, post_id, reaction)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, post_id) DO UPDATE SET reaction = ?
	`, cmd.UserID, cmd.PostID, cmd.Reaction, cmd.Reaction)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert reaction: %w", err)
	}

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"action":   "added",
			"reaction": cmd.Reaction,
		},
	}, nil
}

// Handle processes ReactToCommentCommand
func (h *PostCommandHandler) ReactToComment(cmd ReactToCommentCommand) (*CommandResult, error) {
	// Validation
	if err := h.validateReaction(cmd.Reaction); err != nil {
		return &CommandResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Check if reaction already exists
	var existingReaction sql.NullString
	err := h.db.QueryRow(
		"SELECT reaction FROM comment_reactions WHERE user_id = ? AND comment_id = ?",
		cmd.UserID, cmd.CommentID,
	).Scan(&existingReaction)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing reaction: %w", err)
	}

	// If same reaction, remove it (toggle off)
	if existingReaction.Valid && existingReaction.String == cmd.Reaction {
		_, err := h.db.Exec(
			"DELETE FROM comment_reactions WHERE user_id = ? AND comment_id = ?",
			cmd.UserID, cmd.CommentID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to remove reaction: %w", err)
		}
		return &CommandResult{
			Success: true,
			Data: map[string]interface{}{
				"action": "removed",
			},
		}, nil
	}

	// Upsert reaction
	_, err = h.db.Exec(`
		INSERT INTO comment_reactions (user_id, comment_id, reaction)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, comment_id) DO UPDATE SET reaction = ?
	`, cmd.UserID, cmd.CommentID, cmd.Reaction, cmd.Reaction)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert reaction: %w", err)
	}

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"action":   "added",
			"reaction": cmd.Reaction,
		},
	}, nil
}

// Validation methods

func (h *PostCommandHandler) validateCreatePost(cmd CreatePostCommand) error {
	if cmd.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}
	
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if len(title) < 3 {
		return fmt.Errorf("title must be at least 3 characters")
	}
	if len(title) > 200 {
		return fmt.Errorf("title must be less than 200 characters")
	}

	content := strings.TrimSpace(cmd.Content)
	if content == "" {
		return fmt.Errorf("content is required")
	}
	if len(content) < 10 {
		return fmt.Errorf("content must be at least 10 characters")
	}

	if len(cmd.CategoryIDs) == 0 {
		return fmt.Errorf("at least one category is required")
	}

	// Verify categories exist
	for _, catID := range cmd.CategoryIDs {
		var exists bool
		err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)", catID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to verify category %d: %w", catID, err)
		}
		if !exists {
			return fmt.Errorf("category %d does not exist", catID)
		}
	}

	return nil
}

func (h *PostCommandHandler) validateCreateComment(cmd CreateCommentCommand) error {
	if cmd.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}
	if cmd.PostID <= 0 {
		return fmt.Errorf("invalid post ID")
	}

	content := strings.TrimSpace(cmd.Content)
	if content == "" {
		return fmt.Errorf("content is required")
	}
	if len(content) < 2 {
		return fmt.Errorf("comment must be at least 2 characters")
	}
	if len(content) > 1000 {
		return fmt.Errorf("comment must be less than 1000 characters")
	}

	return nil
}

func (h *PostCommandHandler) validateReaction(reaction string) error {
	if reaction != "like" && reaction != "dislike" {
		return fmt.Errorf("reaction must be 'like' or 'dislike'")
	}
	return nil
}

// GetCategories returns all categories (for form dropdowns)
func (h *PostCommandHandler) GetCategories() ([]models.Category, error) {
	rows, err := h.db.Query("SELECT id, label FROM categories ORDER BY label")
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		if err := rows.Scan(&cat.ID, &cat.Label); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, cat)
	}

	return categories, nil
}
