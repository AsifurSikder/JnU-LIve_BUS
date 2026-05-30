-- Create stops table for route waypoints
CREATE TABLE IF NOT EXISTS stops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    latitude DECIMAL(10, 8) NOT NULL CHECK (latitude >= -90 AND latitude <= 90),
    longitude DECIMAL(11, 8) NOT NULL CHECK (longitude >= -180 AND longitude <= 180),
    stop_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(route_id, stop_order)
);

-- Create indexes for stops table
CREATE INDEX IF NOT EXISTS idx_stops_route_id ON stops(route_id);
CREATE INDEX IF NOT EXISTS idx_stops_route_order ON stops(route_id, stop_order);

-- Add comments to table
COMMENT ON TABLE stops IS 'Stores ordered stops for each route with geographic coordinates';
COMMENT ON COLUMN stops.route_id IS 'Foreign key to routes table with cascade delete';
COMMENT ON COLUMN stops.latitude IS 'Latitude coordinate (-90 to 90)';
COMMENT ON COLUMN stops.longitude IS 'Longitude coordinate (-180 to 180)';
COMMENT ON COLUMN stops.stop_order IS 'Order of stop in route sequence (unique per route)';
