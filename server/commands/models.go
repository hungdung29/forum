package commands

// CreatePostCommand represents a command to create a new post
type CreatePostCommand struct {
	UserID      int      `json:"user_id"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	CategoryIDs []int    `json:"category_ids"`
}

// CreateCommentCommand represents a command to add a comment
type CreateCommentCommand struct {
	UserID  int    `json:"user_id"`
	PostID  int    `json:"post_id"`
	Content string `json:"content"`
}

// ReactToPostCommand represents a command to like/dislike a post
type ReactToPostCommand struct {
	UserID   int    `json:"user_id"`
	PostID   int    `json:"post_id"`
	Reaction string `json:"reaction"` // "like" or "dislike"
}

// ReactToCommentCommand represents a command to like/dislike a comment
type ReactToCommentCommand struct {
	UserID    int    `json:"user_id"`
	CommentID int    `json:"comment_id"`
	Reaction  string `json:"reaction"` // "like" or "dislike"
}

// RegisterUserCommand represents a command to register a new user
type RegisterUserCommand struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginCommand represents a command to authenticate a user
type LoginCommand struct {
	EmailOrUsername string `json:"email_or_username"`
	Password        string `json:"password"`
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
