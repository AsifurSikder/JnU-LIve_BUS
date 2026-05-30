-- Create users table for driver and admin authentication
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('driver', 'admin')),
    full_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Add comment to table
COMMENT ON TABLE users IS 'Stores driver and admin user credentials with bcrypt-hashed passwords';
COMMENT ON COLUMN users.role IS 'User role: driver or admin';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt-hashed password with cost factor 12';
