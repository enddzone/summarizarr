CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE,
    number TEXT,
    name TEXT
);

CREATE TABLE IF NOT EXISTS groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id TEXT UNIQUE,
    name TEXT
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp INTEGER,
    server_received_timestamp INTEGER,
    server_delivered_timestamp INTEGER,
    message_text TEXT,
    message_type TEXT DEFAULT 'message', -- 'message', 'reaction', 'quote'
    
    -- Quote fields
    quote_id INTEGER,
    quote_author_uuid TEXT,
    quote_text TEXT,
    
    -- Reaction fields
    is_reaction BOOLEAN DEFAULT FALSE,
    reaction_emoji TEXT,
    reaction_target_author_uuid TEXT,
    reaction_target_timestamp INTEGER,
    reaction_is_remove BOOLEAN DEFAULT FALSE,
    
    user_id INTEGER,
    group_id INTEGER,
    FOREIGN KEY (user_id) REFERENCES users (id),
    FOREIGN KEY (group_id) REFERENCES groups (id)
);

CREATE TABLE IF NOT EXISTS summaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER,
    summary_text TEXT,
    start_timestamp INTEGER,
    end_timestamp INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES groups (id)
);

-- Authentication users table (separate from Signal users)
CREATE TABLE IF NOT EXISTS auth_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_auth_users_email ON auth_users(email);

-- Sessions table for SCS (session management)
CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry);

-- Rotation metadata tables removed: encryption_info and encryption_rotation_log (kept only on existing installs)