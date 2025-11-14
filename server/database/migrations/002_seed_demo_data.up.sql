-- Insert demo users with hashed passwords (password: "password123")
-- Note: In production, use bcrypt to hash these properly
INSERT INTO users (email, username, password) VALUES
('alice@example.com', 'alice', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'),
('bob@example.com', 'bob', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'),
('charlie@example.com', 'charlie', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'),
('diana@example.com', 'diana', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'),
('eve@example.com', 'eve', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy');

-- Insert categories
INSERT INTO categories (label) VALUES
('Technology'),
('Science'),
('Programming'),
('Design'),
('General Discussion');

-- Insert posts
INSERT INTO posts (user_id, title, content) VALUES
(1, 'Welcome to the Forum!', 'This is a test post to get started. Feel free to explore the forum features!'),
(2, 'Discussing Go Best Practices', 'What are your favorite Go patterns and why? Let''s discuss clean architecture and testing strategies.'),
(3, 'Docker Multi-Stage Builds', 'Multi-stage Docker builds can significantly reduce image size. Here are some tips...'),
(1, 'Health Check Implementation', 'Just implemented a health check endpoint. It monitors database, disk space, and memory usage.'),
(4, 'Rate Limiting Strategies', 'How do you implement rate limiting in your applications? Token bucket vs sliding window?');

-- Link posts with categories
INSERT INTO post_category (post_id, category_id) VALUES
(1, 5),  -- Welcome -> General
(2, 3),  -- Go Best Practices -> Programming
(3, 1),  -- Docker -> Technology
(4, 3),  -- Health Check -> Programming
(5, 1);  -- Rate Limiting -> Technology

-- Insert comments
INSERT INTO comments (user_id, post_id, content) VALUES
(2, 1, 'Great to be here! Looking forward to interesting discussions.'),
(3, 1, 'Nice forum setup!'),
(1, 2, 'I love using the repository pattern for database access.'),
(4, 2, 'Don''t forget about context.Context for cancellation!'),
(5, 3, 'Multi-stage builds saved us 80% in image size!'),
(2, 4, 'Health checks are crucial for Kubernetes deployments.'),
(3, 5, 'We use token bucket with Redis for distributed rate limiting.');

-- Insert some reactions
INSERT INTO post_reactions (user_id, post_id, reaction) VALUES
(2, 1, 'like'),
(3, 1, 'like'),
(4, 1, 'like'),
(1, 2, 'like'),
(3, 2, 'like'),
(2, 3, 'like'),
(5, 4, 'like'),
(1, 5, 'like');

INSERT INTO comment_reactions (user_id, comment_id, reaction) VALUES
(1, 1, 'like'),
(2, 2, 'like'),
(3, 3, 'like'),
(5, 4, 'like'),
(1, 5, 'like');
