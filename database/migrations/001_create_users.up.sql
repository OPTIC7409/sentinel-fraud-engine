-- Create users table for dashboard authentication
CREATE TABLE IF NOT EXISTS users (
    id              SERIAL PRIMARY KEY,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    role            VARCHAR(20) NOT NULL CHECK (role IN ('analyst', 'admin')),
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login      TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);

COMMENT ON TABLE users IS 'Dashboard users (fraud analysts and admins)';
COMMENT ON COLUMN users.password_hash IS 'bcrypt hash with cost factor 12';
COMMENT ON COLUMN users.role IS 'analyst: read-only access, admin: full access';
