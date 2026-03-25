-- Seed initial users for testing
-- Password for both is: sentinel123 (bcrypt hash with cost 12)
INSERT INTO users (email, password_hash, full_name, role) VALUES
('admin@sentinel.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIq.Kmykq2', 'Admin User', 'admin'),
('analyst@sentinel.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIq.Kmykq2', 'Alice Johnson', 'analyst'),
('bob.smith@sentinel.com', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIq.Kmykq2', 'Bob Smith', 'analyst')
ON CONFLICT (email) DO NOTHING;
