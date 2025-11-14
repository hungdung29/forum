package queries

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// CachedPostQueryService wraps PostQueryService with caching
type CachedPostQueryService struct {
	queryService *PostQueryService
	cache        *QueryCache
}

// QueryCache provides simple in-memory caching for queries
type QueryCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
	ttl   time.Duration
}

type cacheItem struct {
	data      interface{}
	expiresAt time.Time
}

// NewQueryCache creates a new query cache
func NewQueryCache(ttl time.Duration) *QueryCache {
	cache := &QueryCache{
		items: make(map[string]*cacheItem),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves an item from cache
func (c *QueryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.data, true
}

// Set stores an item in cache
func (c *QueryCache) Set(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes items with matching key prefix
func (c *QueryCache) Invalidate(keyPrefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.items {
		if len(keyPrefix) == 0 || key[:len(keyPrefix)] == keyPrefix {
			delete(c.items, key)
		}
	}
}

// cleanup removes expired items periodically
func (c *QueryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// NewCachedPostQueryService creates a cached query service
func NewCachedPostQueryService(db *sql.DB, cacheTTL time.Duration) *CachedPostQueryService {
	return &CachedPostQueryService{
		queryService: NewPostQueryService(db),
		cache:        NewQueryCache(cacheTTL),
	}
}

// GetAllPosts with caching
func (s *CachedPostQueryService) GetAllPosts(userID int) ([]PostListItem, error) {
	cacheKey := fmt.Sprintf("posts_all_user_%d", userID)

	// Try cache first
	if cached, found := s.cache.Get(cacheKey); found {
		return cached.([]PostListItem), nil
	}

	// Query database
	posts, err := s.queryService.GetAllPosts(userID)
	if err != nil {
		return nil, err
	}

	// Cache result
	s.cache.Set(cacheKey, posts)
	return posts, nil
}

// GetPostByID with caching
func (s *CachedPostQueryService) GetPostByID(postID, userID int) (*PostDetail, error) {
	cacheKey := fmt.Sprintf("post_%d_user_%d", postID, userID)

	// Try cache first
	if cached, found := s.cache.Get(cacheKey); found {
		return cached.(*PostDetail), nil
	}

	// Query database
	post, err := s.queryService.GetPostByID(postID, userID)
	if err != nil {
		return nil, err
	}

	// Cache result
	s.cache.Set(cacheKey, post)
	return post, nil
}

// GetPostsByCategory with caching
func (s *CachedPostQueryService) GetPostsByCategory(categoryID, userID int) ([]PostListItem, error) {
	cacheKey := fmt.Sprintf("posts_cat_%d_user_%d", categoryID, userID)

	// Try cache first
	if cached, found := s.cache.Get(cacheKey); found {
		return cached.([]PostListItem), nil
	}

	// Query database
	posts, err := s.queryService.GetPostsByCategory(categoryID, userID)
	if err != nil {
		return nil, err
	}

	// Cache result
	s.cache.Set(cacheKey, posts)
	return posts, nil
}

// GetUserCreatedPosts with caching
func (s *CachedPostQueryService) GetUserCreatedPosts(userID int) ([]PostListItem, error) {
	cacheKey := fmt.Sprintf("posts_created_user_%d", userID)

	// Try cache first
	if cached, found := s.cache.Get(cacheKey); found {
		return cached.([]PostListItem), nil
	}

	// Query database
	posts, err := s.queryService.GetUserCreatedPosts(userID)
	if err != nil {
		return nil, err
	}

	// Cache result
	s.cache.Set(cacheKey, posts)
	return posts, nil
}

// GetUserLikedPosts with caching
func (s *CachedPostQueryService) GetUserLikedPosts(userID int) ([]PostListItem, error) {
	cacheKey := fmt.Sprintf("posts_liked_user_%d", userID)

	// Try cache first
	if cached, found := s.cache.Get(cacheKey); found {
		return cached.([]PostListItem), nil
	}

	// Query database
	posts, err := s.queryService.GetUserLikedPosts(userID)
	if err != nil {
		return nil, err
	}

	// Cache result
	s.cache.Set(cacheKey, posts)
	return posts, nil
}

// GetAllCategories with caching
func (s *CachedPostQueryService) GetAllCategories() ([]CategorySummary, error) {
	cacheKey := "categories_all"

	// Try cache first
	if cached, found := s.cache.Get(cacheKey); found {
		return cached.([]CategorySummary), nil
	}

	// Query database
	categories, err := s.queryService.GetAllCategories()
	if err != nil {
		return nil, err
	}

	// Cache result (categories change rarely, so cache longer)
	s.cache.Set(cacheKey, categories)
	return categories, nil
}

// InvalidatePostCache invalidates all post-related cache entries
func (s *CachedPostQueryService) InvalidatePostCache() {
	s.cache.Invalidate("posts_")
	s.cache.Invalidate("post_")
}

// InvalidateUserCache invalidates user-specific cache entries
func (s *CachedPostQueryService) InvalidateUserCache(userID int) {
	s.cache.Invalidate(fmt.Sprintf("user_%d", userID))
	s.cache.Invalidate(fmt.Sprintf("posts_created_user_%d", userID))
	s.cache.Invalidate(fmt.Sprintf("posts_liked_user_%d", userID))
}
