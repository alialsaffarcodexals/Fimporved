-- Schema for forum MVP database

-- Users & sessions tables store credentials and login sessions.
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY,              -- unique user ID
    email TEXT NOT NULL UNIQUE,          -- used for login
    username TEXT NOT NULL UNIQUE,       -- public name
    password_hash TEXT NOT NULL,         -- bcrypt hash
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP -- when account was created
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,              -- UUID
    user_id INTEGER NOT NULL UNIQUE,  -- only one session per user
    expires_at DATETIME NOT NULL,     -- session expiry
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Content tables hold posts and comments.
CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY,                       -- post identifier
    user_id INTEGER NOT NULL,                     -- author
    title TEXT NOT NULL,                          -- post title
    body TEXT NOT NULL,                           -- post body text
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, -- creation time
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY,                       -- comment identifier
    post_id INTEGER NOT NULL,                     -- parent post
    user_id INTEGER NOT NULL,                     -- author
    body TEXT NOT NULL,                           -- comment text
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, -- when posted
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Categories let posts be organized.
CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY,           -- category id
    name TEXT NOT NULL UNIQUE         -- category name
);

CREATE TABLE IF NOT EXISTS post_categories (
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, category_id), -- composite key ensures uniqueness
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE CASCADE
);

-- Likes (posts & comments) track upvotes and downvotes.
CREATE TABLE IF NOT EXISTS likes (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,                                       -- who voted
    target_type TEXT NOT NULL CHECK (target_type IN ('post','comment')), -- post or comment
    target_id INTEGER NOT NULL,                                     -- target id
    value INTEGER NOT NULL CHECK (value IN (-1,1)),                 -- up or down
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, target_type, target_id),                       -- one vote per user per item
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Seed a few categories (idempotent)
INSERT INTO categories (name)
    SELECT 'General' WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name='General');
INSERT INTO categories (name)
    SELECT 'Help' WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name='Help');
INSERT INTO categories (name)
    SELECT 'Off-topic' WHERE NOT EXISTS (SELECT 1 FROM categories WHERE name='Off-topic');
