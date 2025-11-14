package queries

import "time"

// PostListItem represents a post in list view (homepage, category page)
type PostListItem struct {
	ID              int       `json:"id"`
	Title           string    `json:"title"`
	ContentPreview  string    `json:"content_preview"` // First 200 chars
	AuthorID        int       `json:"author_id"`
	AuthorUsername  string    `json:"author_username"`
	CreatedAt       time.Time `json:"created_at"`
	CommentCount    int       `json:"comment_count"`
	LikeCount       int       `json:"like_count"`
	DislikeCount    int       `json:"dislike_count"`
	Categories      []string  `json:"categories"`
	UserHasLiked    bool      `json:"user_has_liked"`
	UserHasDisliked bool      `json:"user_has_disliked"`
}

// PostDetail represents full post details for post view page
type PostDetail struct {
	ID              int       `json:"id"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	AuthorID        int       `json:"author_id"`
	AuthorUsername  string    `json:"author_username"`
	CreatedAt       time.Time `json:"created_at"`
	Categories      []string  `json:"categories"`
	LikeCount       int       `json:"like_count"`
	DislikeCount    int       `json:"dislike_count"`
	UserHasLiked    bool      `json:"user_has_liked"`
	UserHasDisliked bool      `json:"user_has_disliked"`
	Comments        []CommentDetail `json:"comments"`
}

// CommentDetail represents a comment with author and reactions
type CommentDetail struct {
	ID              int       `json:"id"`
	PostID          int       `json:"post_id"`
	Content         string    `json:"content"`
	AuthorID        int       `json:"author_id"`
	AuthorUsername  string    `json:"author_username"`
	CreatedAt       time.Time `json:"created_at"`
	LikeCount       int       `json:"like_count"`
	DislikeCount    int       `json:"dislike_count"`
	UserHasLiked    bool      `json:"user_has_liked"`
	UserHasDisliked bool      `json:"user_has_disliked"`
}

// UserPostsSummary for "My Posts" page
type UserPostsSummary struct {
	TotalPosts      int            `json:"total_posts"`
	TotalComments   int            `json:"total_comments"`
	TotalLikes      int            `json:"total_likes"`
	RecentPosts     []PostListItem `json:"recent_posts"`
}

// CategorySummary for category listing
type CategorySummary struct {
	ID        int    `json:"id"`
	Label     string `json:"label"`
	PostCount int    `json:"post_count"`
}
