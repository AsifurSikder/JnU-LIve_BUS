-- Create routes table for managing bus routes
CREATE TABLE IF NOT EXISTS routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index for routes table
CREATE INDEX IF NOT EXISTS idx_routes_name ON routes(name);

-- Add comment to table
COMMENT ON TABLE routes IS 'Stores bus routes with unique names';
COMMENT ON COLUMN routes.name IS 'Unique route name (1-100 characters)';
