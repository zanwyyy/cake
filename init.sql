-- ========================
--  Users table
-- ========================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    password TEXT NOT NULL
);

-- ========================
--  Sessions table
-- ========================
CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_token TEXT,
    refresh_token TEXT,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- ========================
--  Indexes
-- ========================
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- ========================
--  Demo seed data (optional)
-- ========================
INSERT INTO users (id, name, balance, password)
VALUES 
    (1, 'alice', 10000, 'password123'),
    (2, 'bob', 5000, 'password456'),
    (3, 'bb', 5000, 'password')
ON CONFLICT (id) DO NOTHING;

