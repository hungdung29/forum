package queries

import (
	"database/sql"
	"fmt"
	"strings"
)

// PostQueryService handles all read operations for posts
type PostQueryService struct {
	db *sql.DB
}

// NewPostQueryService creates a new query service
func NewPostQueryService(db *sql.DB) *PostQueryService {
	return &PostQueryService{db: db}
}

// GetAllPosts retrieves all posts with aggregated data (homepage)
func (s *PostQueryService) GetAllPosts(userID int) ([]PostListItem, error) {
	query := `
		SELECT 
			p.id,
			p.title,
			SUBSTR(p.content, 1, 200) as content_preview,
			p.user_id,
			u.username,
			p.created_at,
			COUNT(DISTINCT c.id) as comment_count,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'like' THEN pr.user_id END) as like_count,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'dislike' THEN pr.user_id END) as dislike_count,
			GROUP_CONCAT(DISTINCT cat.label) as categories,
			MAX(CASE WHEN pr.user_id = ? AND pr.reaction = 'like' THEN 1 ELSE 0 END) as user_has_liked,
			MAX(CASE WHEN pr.user_id = ? AND pr.reaction = 'dislike' THEN 1 ELSE 0 END) as user_has_disliked
		FROM posts p
		LEFT JOIN users u ON p.user_id = u.id
		LEFT JOIN comments c ON p.id = c.post_id
		LEFT JOIN post_reactions pr ON p.id = pr.post_id
		LEFT JOIN post_category pc ON p.id = pc.post_id
		LEFT JOIN categories cat ON pc.category_id = cat.id
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := s.db.Query(query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	var posts []PostListItem
	for rows.Next() {
		var post PostListItem
		var categoriesStr sql.NullString
		var contentPreview sql.NullString

		err := rows.Scan(
			&post.ID,
			&post.Title,
			&contentPreview,
			&post.AuthorID,
			&post.AuthorUsername,
			&post.CreatedAt,
			&post.CommentCount,
			&post.LikeCount,
			&post.DislikeCount,
			&categoriesStr,
			&post.UserHasLiked,
			&post.UserHasDisliked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		if contentPreview.Valid {
			post.ContentPreview = contentPreview.String
			if len(post.ContentPreview) == 200 {
				post.ContentPreview += "..."
			}
		}

		if categoriesStr.Valid && categoriesStr.String != "" {
			post.Categories = strings.Split(categoriesStr.String, ",")
		} else {
			post.Categories = []string{}
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// GetPostByID retrieves full post details with comments
func (s *PostQueryService) GetPostByID(postID, userID int) (*PostDetail, error) {
	// Get post details
	query := `
		SELECT 
			p.id,
			p.title,
			p.content,
			p.user_id,
			u.username,
			p.created_at,
			GROUP_CONCAT(DISTINCT cat.label) as categories,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'like' THEN pr.user_id END) as like_count,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'dislike' THEN pr.user_id END) as dislike_count,
			MAX(CASE WHEN pr.user_id = ? AND pr.reaction = 'like' THEN 1 ELSE 0 END) as user_has_liked,
			MAX(CASE WHEN pr.user_id = ? AND pr.reaction = 'dislike' THEN 1 ELSE 0 END) as user_has_disliked
		FROM posts p
		LEFT JOIN users u ON p.user_id = u.id
		LEFT JOIN post_reactions pr ON p.id = pr.post_id
		LEFT JOIN post_category pc ON p.id = pc.post_id
		LEFT JOIN categories cat ON pc.category_id = cat.id
		WHERE p.id = ?
		GROUP BY p.id
	`

	var post PostDetail
	var categoriesStr sql.NullString

	err := s.db.QueryRow(query, userID, userID, postID).Scan(
		&post.ID,
		&post.Title,
		&post.Content,
		&post.AuthorID,
		&post.AuthorUsername,
		&post.CreatedAt,
		&categoriesStr,
		&post.LikeCount,
		&post.DislikeCount,
		&post.UserHasLiked,
		&post.UserHasDisliked,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post not found")
		}
		return nil, fmt.Errorf("failed to query post: %w", err)
	}

	if categoriesStr.Valid && categoriesStr.String != "" {
		post.Categories = strings.Split(categoriesStr.String, ",")
	} else {
		post.Categories = []string{}
	}

	// Get comments
	comments, err := s.getCommentsByPostID(postID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	post.Comments = comments

	return &post, nil
}

// getCommentsByPostID retrieves all comments for a post
func (s *PostQueryService) getCommentsByPostID(postID, userID int) ([]CommentDetail, error) {
	query := `
		SELECT 
			c.id,
			c.post_id,
			c.content,
			c.user_id,
			u.username,
			c.created_at,
			COUNT(DISTINCT CASE WHEN cr.reaction = 'like' THEN cr.user_id END) as like_count,
			COUNT(DISTINCT CASE WHEN cr.reaction = 'dislike' THEN cr.user_id END) as dislike_count,
			MAX(CASE WHEN cr.user_id = ? AND cr.reaction = 'like' THEN 1 ELSE 0 END) as user_has_liked,
			MAX(CASE WHEN cr.user_id = ? AND cr.reaction = 'dislike' THEN 1 ELSE 0 END) as user_has_disliked
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN comment_reactions cr ON c.id = cr.comment_id
		WHERE c.post_id = ?
		GROUP BY c.id
		ORDER BY c.created_at ASC
	`

	rows, err := s.db.Query(query, userID, userID, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var comments []CommentDetail
	for rows.Next() {
		var comment CommentDetail
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.Content,
			&comment.AuthorID,
			&comment.AuthorUsername,
			&comment.CreatedAt,
			&comment.LikeCount,
			&comment.DislikeCount,
			&comment.UserHasLiked,
			&comment.UserHasDisliked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

// GetPostsByCategory retrieves posts filtered by category
func (s *PostQueryService) GetPostsByCategory(categoryID, userID int) ([]PostListItem, error) {
	query := `
		SELECT 
			p.id,
			p.title,
			SUBSTR(p.content, 1, 200) as content_preview,
			p.user_id,
			u.username,
			p.created_at,
			COUNT(DISTINCT c.id) as comment_count,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'like' THEN pr.user_id END) as like_count,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'dislike' THEN pr.user_id END) as dislike_count,
			GROUP_CONCAT(DISTINCT cat.label) as categories,
			MAX(CASE WHEN pr.user_id = ? AND pr.reaction = 'like' THEN 1 ELSE 0 END) as user_has_liked,
			MAX(CASE WHEN pr.user_id = ? AND pr.reaction = 'dislike' THEN 1 ELSE 0 END) as user_has_disliked
		FROM posts p
		LEFT JOIN users u ON p.user_id = u.id
		LEFT JOIN comments c ON p.id = c.post_id
		LEFT JOIN post_reactions pr ON p.id = pr.post_id
		LEFT JOIN post_category pc ON p.id = pc.post_id
		LEFT JOIN categories cat ON pc.category_id = cat.id
		WHERE p.id IN (
			SELECT post_id FROM post_category WHERE category_id = ?
		)
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := s.db.Query(query, userID, userID, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts by category: %w", err)
	}
	defer rows.Close()

	var posts []PostListItem
	for rows.Next() {
		var post PostListItem
		var categoriesStr sql.NullString
		var contentPreview sql.NullString

		err := rows.Scan(
			&post.ID,
			&post.Title,
			&contentPreview,
			&post.AuthorID,
			&post.AuthorUsername,
			&post.CreatedAt,
			&post.CommentCount,
			&post.LikeCount,
			&post.DislikeCount,
			&categoriesStr,
			&post.UserHasLiked,
			&post.UserHasDisliked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		if contentPreview.Valid {
			post.ContentPreview = contentPreview.String
			if len(post.ContentPreview) == 200 {
				post.ContentPreview += "..."
			}
		}

		if categoriesStr.Valid && categoriesStr.String != "" {
			post.Categories = strings.Split(categoriesStr.String, ",")
		} else {
			post.Categories = []string{}
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// GetUserCreatedPosts retrieves posts created by a user
func (s *PostQueryService) GetUserCreatedPosts(userID int) ([]PostListItem, error) {
	query := `
		SELECT 
			p.id,
			p.title,
			SUBSTR(p.content, 1, 200) as content_preview,
			p.user_id,
			u.username,
			p.created_at,
			COUNT(DISTINCT c.id) as comment_count,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'like' THEN pr.user_id END) as like_count,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'dislike' THEN pr.user_id END) as dislike_count,
			GROUP_CONCAT(DISTINCT cat.label) as categories,
			1 as user_has_liked,
			0 as user_has_disliked
		FROM posts p
		LEFT JOIN users u ON p.user_id = u.id
		LEFT JOIN comments c ON p.id = c.post_id
		LEFT JOIN post_reactions pr ON p.id = pr.post_id
		LEFT JOIN post_category pc ON p.id = pc.post_id
		LEFT JOIN categories cat ON pc.category_id = cat.id
		WHERE p.user_id = ?
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user posts: %w", err)
	}
	defer rows.Close()

	var posts []PostListItem
	for rows.Next() {
		var post PostListItem
		var categoriesStr sql.NullString
		var contentPreview sql.NullString

		err := rows.Scan(
			&post.ID,
			&post.Title,
			&contentPreview,
			&post.AuthorID,
			&post.AuthorUsername,
			&post.CreatedAt,
			&post.CommentCount,
			&post.LikeCount,
			&post.DislikeCount,
			&categoriesStr,
			&post.UserHasLiked,
			&post.UserHasDisliked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		if contentPreview.Valid {
			post.ContentPreview = contentPreview.String
			if len(post.ContentPreview) == 200 {
				post.ContentPreview += "..."
			}
		}

		if categoriesStr.Valid && categoriesStr.String != "" {
			post.Categories = strings.Split(categoriesStr.String, ",")
		} else {
			post.Categories = []string{}
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// GetUserLikedPosts retrieves posts liked by a user
func (s *PostQueryService) GetUserLikedPosts(userID int) ([]PostListItem, error) {
	query := `
		SELECT 
			p.id,
			p.title,
			SUBSTR(p.content, 1, 200) as content_preview,
			p.user_id,
			u.username,
			p.created_at,
			COUNT(DISTINCT c.id) as comment_count,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'like' THEN pr.user_id END) as like_count,
			COUNT(DISTINCT CASE WHEN pr.reaction = 'dislike' THEN pr.user_id END) as dislike_count,
			GROUP_CONCAT(DISTINCT cat.label) as categories,
			1 as user_has_liked,
			0 as user_has_disliked
		FROM posts p
		LEFT JOIN users u ON p.user_id = u.id
		LEFT JOIN comments c ON p.id = c.post_id
		LEFT JOIN post_reactions pr ON p.id = pr.post_id
		LEFT JOIN post_category pc ON p.id = pc.post_id
		LEFT JOIN categories cat ON pc.category_id = cat.id
		WHERE p.id IN (
			SELECT post_id FROM post_reactions WHERE user_id = ? AND reaction = 'like'
		)
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query liked posts: %w", err)
	}
	defer rows.Close()

	var posts []PostListItem
	for rows.Next() {
		var post PostListItem
		var categoriesStr sql.NullString
		var contentPreview sql.NullString

		err := rows.Scan(
			&post.ID,
			&post.Title,
			&contentPreview,
			&post.AuthorID,
			&post.AuthorUsername,
			&post.CreatedAt,
			&post.CommentCount,
			&post.LikeCount,
			&post.DislikeCount,
			&categoriesStr,
			&post.UserHasLiked,
			&post.UserHasDisliked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		if contentPreview.Valid {
			post.ContentPreview = contentPreview.String
			if len(post.ContentPreview) == 200 {
				post.ContentPreview += "..."
			}
		}

		if categoriesStr.Valid && categoriesStr.String != "" {
			post.Categories = strings.Split(categoriesStr.String, ",")
		} else {
			post.Categories = []string{}
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// GetAllCategories retrieves all categories with post counts
func (s *PostQueryService) GetAllCategories() ([]CategorySummary, error) {
	query := `
		SELECT 
			c.id,
			c.label,
			COUNT(DISTINCT pc.post_id) as post_count
		FROM categories c
		LEFT JOIN post_category pc ON c.id = pc.category_id
		GROUP BY c.id
		ORDER BY c.label ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []CategorySummary
	for rows.Next() {
		var cat CategorySummary
		err := rows.Scan(&cat.ID, &cat.Label, &cat.PostCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, cat)
	}

	return categories, nil
}
