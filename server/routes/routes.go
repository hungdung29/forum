package routes

import (
	"database/sql"
	"net/http"
	"time"

	"forum/server/controllers"
	"forum/server/middleware"
)

func Routes(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	// Initialize rate limiter
	limiter := middleware.NewRateLimiter()
	
	// Rate limit configurations
	publicLimit := middleware.RateLimit(limiter, 100, time.Minute)     // 100 req/min for public
	loginLimit := middleware.RateLimit(limiter, 5, time.Minute)        // 5 req/min for login (brute-force protection)
	createLimit := middleware.RateLimit(limiter, 10, time.Minute)      // 10 req/min for creates (spam protection)

	// serve static files (no rate limit needed)
	mux.HandleFunc("/assets/", controllers.ServeStaticFiles)

	// Health check endpoint (no auth, no rate limit - used by load balancers)
	mux.HandleFunc("/health", controllers.HealthCheck(db))

	// Public routes with rate limiting
	mux.HandleFunc("/", publicLimit(func(w http.ResponseWriter, r *http.Request) {
		controllers.IndexPosts(w, r, db)
	}))
	
	mux.HandleFunc("/category/{id}", publicLimit(func(w http.ResponseWriter, r *http.Request) {
		controllers.IndexPostsByCategory(w, r, db)
	}))
	
	mux.HandleFunc("/post/{id}", publicLimit(func(w http.ResponseWriter, r *http.Request) {
		controllers.ShowPost(w, r, db)
	}))

	// Auth routes - strict rate limiting to prevent brute force
	mux.HandleFunc("/login", loginLimit(func(w http.ResponseWriter, r *http.Request) {
		controllers.GetLoginPage(w, r, db)
	}))
	
	mux.HandleFunc("/signin", loginLimit(middleware.Sanitize(func(w http.ResponseWriter, r *http.Request) {
		controllers.Signin(w, r, db)
	})))
	
	mux.HandleFunc("/register", loginLimit(func(w http.ResponseWriter, r *http.Request) {
		controllers.GetRegisterPage(w, r, db)
	}))
	
	mux.HandleFunc("/signup", loginLimit(middleware.Sanitize(func(w http.ResponseWriter, r *http.Request) {
		controllers.Signup(w, r, db)
	})))
	
	mux.HandleFunc("/logout", publicLimit(func(w http.ResponseWriter, r *http.Request) {
		controllers.Logout(w, r, db)
	}))

	// Protected routes - moderate rate limiting + input sanitization
	mux.HandleFunc("/mycreatedposts", publicLimit(func(w http.ResponseWriter, r *http.Request) {
		controllers.MyCreatedPosts(w, r, db)
	}))
	
	mux.HandleFunc("/mylikedposts", publicLimit(func(w http.ResponseWriter, r *http.Request) {
		controllers.MyLikedPosts(w, r, db)
	}))
	
	mux.HandleFunc("/post/create", publicLimit(func(w http.ResponseWriter, r *http.Request) {
		controllers.GetPostCreationForm(w, r, db)
	}))

	// Create/mutate routes - strict rate limiting + sanitization
	mux.HandleFunc("/post/createpost", createLimit(middleware.Sanitize(func(w http.ResponseWriter, r *http.Request) {
		controllers.CreatePost(w, r, db)
	})))
	
	mux.HandleFunc("/post/addcommentREQ", createLimit(middleware.Sanitize(func(w http.ResponseWriter, r *http.Request) {
		controllers.CreateComment(w, r, db)
	})))

	mux.HandleFunc("/post/postreaction", createLimit(middleware.Sanitize(func(w http.ResponseWriter, r *http.Request) {
		controllers.ReactToPost(w, r, db)
	})))

	mux.HandleFunc("/post/commentreaction", createLimit(middleware.Sanitize(func(w http.ResponseWriter, r *http.Request) {
		controllers.ReactToComment(w, r, db)
	})))

	return mux
}
