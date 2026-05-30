# Design Document

## Overview

The University Bus Tracker is a real-time location tracking system built on a microservices architecture using Go and the Gin web framework. The system enables students to view live bus locations on a map, allows drivers to broadcast GPS coordinates continuously, and provides administrators with tools to manage routes and stops through a web panel.

### System Goals

- **Real-time tracking**: Display bus locations with sub-3-second latency from GPS update to map display
- **High availability**: Support 30 concurrent buses and 300 concurrent rider connections
- **Cost efficiency**: Operate within $0-$5/month using free-tier infrastructure (Railway, Neon PostgreSQL, Upstash Redis)
- **Offline resilience**: Gracefully handle network interruptions with last-known-position fallback
- **Security**: JWT-based authentication with role-based access control

### Technology Stack

**Backend Services:**
- Go 1.21+ with Gin web framework
- PostgreSQL 15 on Neon free tier (persistent storage)
- Redis 7 on Upstash free tier (caching and pub/sub)
- WebSocket (gorilla/websocket) for real-time communication

**Frontend Applications:**
- Android app (Java/Kotlin) for riders
- Android app (Java/Kotlin) for drivers
- React 18 web app for admin panel

**Deployment:**
- Railway for Go container hosting
- Docker for containerization
- GitHub Actions for CI/CD

---

## Architecture

### Microservices Overview

The system consists of four primary services:

1. **API Gateway**: Routes HTTP/WebSocket requests to appropriate microservices, enforces authentication and CORS
2. **Auth Service**: Handles driver and admin authentication, issues JWT tokens
3. **Location Service**: Processes GPS updates, maintains last-known positions, broadcasts to WebSocket clients
4. **Route Service**: Manages routes, stops, and bus-to-route assignments


### Architecture Diagram

```mermaid
graph TB
    subgraph "Client Applications"
        RiderApp[Rider App<br/>Android]
        DriverApp[Driver App<br/>Android]
        AdminPanel[Admin Panel<br/>React Web]
    end
    
    subgraph "API Layer"
        Gateway[API Gateway<br/>Go + Gin<br/>Port 8080]
    end
    
    subgraph "Microservices"
        AuthSvc[Auth Service<br/>Go + Gin<br/>Port 8081]
        LocationSvc[Location Service<br/>Go + Gin<br/>Port 8082]
        RouteSvc[Route Service<br/>Go + Gin<br/>Port 8083]
    end
    
    subgraph "Data Layer"
        Postgres[(PostgreSQL<br/>Neon Free Tier)]
        Redis[(Redis<br/>Upstash Free Tier)]
    end
    
    RiderApp -->|HTTP/WS| Gateway
    DriverApp -->|HTTP/WS| Gateway
    AdminPanel -->|HTTP/WS| Gateway
    
    Gateway -->|/auth/*| AuthSvc
    Gateway -->|/location/*| LocationSvc
    Gateway -->|/routes/*| RouteSvc
    Gateway -->|WebSocket| LocationSvc
    
    AuthSvc -->|Read/Write| Postgres
    LocationSvc -->|Pub/Sub| Redis
    LocationSvc -->|Read/Write| Redis
    RouteSvc -->|Read/Write| Postgres
    RouteSvc -->|Publish| Redis
```


### Service Communication Patterns

**Synchronous Communication (HTTP):**
- Client → API Gateway → Microservices
- Used for: Authentication, route CRUD operations, initial data fetching

**Asynchronous Communication (Redis Pub/Sub):**
- Location Service publishes GPS updates to Redis channel `bus:location:updates`
- Route Service publishes route changes to Redis channel `route:updates`
- Location Service subscribes to both channels to maintain consistency

**Real-time Communication (WebSocket):**
- Rider App ↔ API Gateway ↔ Location Service
- Admin Panel ↔ API Gateway ↔ Location Service
- Driver App → API Gateway → Location Service (optional, primarily uses HTTP POST)

### Service Responsibilities

**API Gateway:**
- Request routing based on URL path prefix
- JWT validation and role-based access control
- CORS enforcement
- Rate limiting for failed authentication attempts
- WebSocket proxy to Location Service
- Health check aggregation

**Auth Service:**
- Driver and admin login
- JWT generation with role-based claims
- Password verification (bcrypt)
- Rate limiting for brute-force protection
- Credential storage in PostgreSQL

**Location Service:**
- GPS coordinate ingestion from drivers
- Coordinate validation (range, timestamp, required fields)
- Last-known position caching in Redis (30-minute TTL)
- Real-time broadcast to WebSocket clients
- Connection management (max 30 drivers, 300 riders)
- Bus active/inactive status tracking

**Route Service:**
- Route CRUD operations
- Stop management (ordered list per route)
- Bus-to-route assignment
- Route change notifications via Redis pub/sub
- Validation (unique route names, stop count limits)

---

## Components and Interfaces

### API Gateway

**Port:** 8080  
**Framework:** Go + Gin  
**Dependencies:** Auth Service, Location Service, Route Service

**Endpoints:**

| Method | Path | Forwarded To | Auth Required | Role |
|--------|------|--------------|---------------|------|
| POST | /auth/login | Auth Service | No | - |
| GET | /routes | Route Service | No | - |
| POST | /routes | Route Service | Yes | Admin |
| PUT | /routes/:id | Route Service | Yes | Admin |
| DELETE | /routes/:id | Route Service | Yes | Admin |
| POST | /routes/:id/assign | Route Service | Yes | Admin |
| POST | /location/update | Location Service | Yes | Driver |
| WS | /ws/location | Location Service | No | - |

**Middleware Chain:**
1. CORS validation (registered origins only)
2. Request logging
3. JWT validation (if Authorization header present)
4. Role-based access control
5. Rate limiting (IP-based for failed auth)
6. Route to downstream service

**Configuration:**
```go
type GatewayConfig struct {
    Port              string
    AuthServiceURL    string
    LocationServiceURL string
    RouteServiceURL   string
    AllowedOrigins    []string
    JWTSecret         string
    RateLimitWindow   time.Duration
    RateLimitMax      int
}
```


### Auth Service

**Port:** 8081  
**Framework:** Go + Gin  
**Dependencies:** PostgreSQL

**Endpoints:**

**POST /auth/login**
- **Description:** Authenticate driver or admin and issue JWT
- **Request Body:**
```json
{
  "username": "string (required, 3-50 chars)",
  "password": "string (required, 8-100 chars)",
  "role": "string (required, enum: 'driver' | 'admin')"
}
```
- **Success Response (200):**
```json
{
  "token": "string (JWT)",
  "expiresAt": "string (ISO 8601 timestamp)",
  "role": "string",
  "userId": "string (UUID)"
}
```
- **Error Responses:**
  - 401: Invalid credentials
  - 422: Validation error
  - 429: Rate limit exceeded (5 failed attempts in 60s)
  - 503: Service unavailable

**JWT Claims Structure:**
```json
{
  "sub": "user-uuid",
  "role": "driver|admin",
  "exp": 1234567890,
  "iat": 1234567890
}
```

**Configuration:**
```go
type AuthConfig struct {
    Port              string
    DatabaseURL       string
    JWTSecret         string
    DriverTokenExpiry time.Duration // 12 hours
    AdminTokenExpiry  time.Duration // 8 hours
    BcryptCost        int           // 12
}
```


### Location Service

**Port:** 8082  
**Framework:** Go + Gin  
**Dependencies:** Redis

**Endpoints:**

**POST /location/update**
- **Description:** Receive GPS coordinate from driver
- **Auth:** Required (Driver JWT)
- **Rate Limit:** Max 1 request per 5 seconds per driver session
- **Request Body:**
```json
{
  "busId": "string (UUID, required)",
  "latitude": "number (required, -90 to 90)",
  "longitude": "number (required, -180 to 180)",
  "timestamp": "string (ISO 8601 UTC, required)",
  "accuracy": "number (optional, meters)",
  "speed": "number (optional, m/s)"
}
```
- **Success Response (200):**
```json
{
  "status": "accepted",
  "broadcastedTo": "number (count of WebSocket clients)"
}
```
- **Error Responses:**
  - 401: Invalid or missing JWT
  - 422: Validation error (invalid coordinates, missing fields, stale timestamp)
  - 429: Rate limit exceeded (< 5 seconds since last update)
  - 503: Service unavailable (connection limit reached)

**WebSocket /ws/location**
- **Description:** Real-time location updates for riders and admins
- **Auth:** Optional (no auth required for riders, admin JWT for admin panel)
- **Connection Limit:** 300 concurrent connections
- **Message Format (Server → Client):**
```json
{
  "type": "location_update",
  "busId": "string (UUID)",
  "routeId": "string (UUID)",
  "routeName": "string",
  "latitude": "number",
  "longitude": "number",
  "timestamp": "string (ISO 8601)",
  "driverName": "string (admin only)"
}
```

**Configuration:**
```go
type LocationConfig struct {
    Port                  string
    RedisURL              string
    MaxDriverConnections  int           // 30
    MaxRiderConnections   int           // 300
    LocationTTL           time.Duration // 30 minutes
    BroadcastTimeout      time.Duration // 2 seconds
    MinUpdateInterval     time.Duration // 5 seconds
    MaxTimestampAge       time.Duration // 5 minutes
}
```


### Route Service

**Port:** 8083  
**Framework:** Go + Gin  
**Dependencies:** PostgreSQL, Redis (pub/sub)

**Endpoints:**

**GET /routes**
- **Description:** Fetch all routes with stops
- **Auth:** Not required
- **Success Response (200):**
```json
{
  "routes": [
    {
      "id": "string (UUID)",
      "name": "string",
      "stops": [
        {
          "id": "string (UUID)",
          "name": "string",
          "latitude": "number",
          "longitude": "number",
          "order": "number"
        }
      ],
      "assignedBusId": "string (UUID, nullable)",
      "assignedDriverName": "string (nullable)"
    }
  ]
}
```

**POST /routes**
- **Description:** Create a new route
- **Auth:** Required (Admin JWT)
- **Request Body:**
```json
{
  "name": "string (required, 1-100 chars, unique)",
  "stops": [
    {
      "name": "string (required, 1-100 chars)",
      "latitude": "number (required, -90 to 90)",
      "longitude": "number (required, -180 to 180)"
    }
  ]
}
```
- **Validation:** 2-50 stops required
- **Success Response (201):** Returns created route with assigned IDs
- **Error Responses:**
  - 403: Forbidden (not admin)
  - 409: Conflict (route name already exists)
  - 422: Validation error
  - 500: Database error

**PUT /routes/:id**
- **Description:** Update route name or stops
- **Auth:** Required (Admin JWT)
- **Request Body:** Same as POST /routes
- **Success Response (200):** Returns updated route
- **Error Responses:** Same as POST, plus 404 if route not found

**DELETE /routes/:id**
- **Description:** Delete a route
- **Auth:** Required (Admin JWT)
- **Success Response (204):** No content
- **Error Responses:**
  - 403: Forbidden
  - 404: Route not found
  - 409: Conflict (route has active bus assigned)

**POST /routes/:id/assign**
- **Description:** Assign a bus/driver to a route
- **Auth:** Required (Admin JWT)
- **Request Body:**
```json
{
  "busId": "string (UUID, required)",
  "driverId": "string (UUID, required)"
}
```
- **Success Response (200):**
```json
{
  "routeId": "string (UUID)",
  "busId": "string (UUID)",
  "driverId": "string (UUID)",
  "assignedAt": "string (ISO 8601)"
}
```
- **Error Responses:**
  - 404: Route, bus, or driver not found
  - 422: Validation error

**Configuration:**
```go
type RouteConfig struct {
    Port        string
    DatabaseURL string
    RedisURL    string
}
```

---

## Data Models

### PostgreSQL Schema

**users table:**
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('driver', 'admin')),
    full_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);
```

**routes table:**
```sql
CREATE TABLE routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_routes_name ON routes(name);
```

**stops table:**
```sql
CREATE TABLE stops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    latitude DECIMAL(10, 8) NOT NULL CHECK (latitude >= -90 AND latitude <= 90),
    longitude DECIMAL(11, 8) NOT NULL CHECK (longitude >= -180 AND longitude <= 180),
    stop_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(route_id, stop_order)
);

CREATE INDEX idx_stops_route_id ON stops(route_id);
CREATE INDEX idx_stops_route_order ON stops(route_id, stop_order);
```

**buses table:**
```sql
CREATE TABLE buses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_plate VARCHAR(20) UNIQUE NOT NULL,
    capacity INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

**route_assignments table:**
```sql
CREATE TABLE route_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    bus_id UUID NOT NULL REFERENCES buses(id) ON DELETE CASCADE,
    driver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(route_id),
    UNIQUE(bus_id)
);

CREATE INDEX idx_route_assignments_route ON route_assignments(route_id);
CREATE INDEX idx_route_assignments_bus ON route_assignments(bus_id);
CREATE INDEX idx_route_assignments_driver ON route_assignments(driver_id);
```


### Redis Data Structures

**Last Known Position (Hash):**
```
Key: bus:location:{busId}
TTL: 30 minutes
Fields:
  - latitude: "23.7104"
  - longitude: "90.4074"
  - timestamp: "2024-01-15T10:30:00Z"
  - routeId: "uuid"
  - routeName: "Route A"
  - driverId: "uuid"
  - driverName: "John Doe"
```

**Active Bus Set:**
```
Key: buses:active
Type: Set
Members: busId (UUID)
TTL: None (managed by Location Service)
```

**Driver Session Rate Limit:**
```
Key: ratelimit:driver:{driverId}
Type: String (timestamp of last accepted update)
TTL: 5 seconds
```

**Failed Auth Attempts (IP-based):**
```
Key: ratelimit:auth:{ip}
Type: String (count)
TTL: 60 seconds
```

**Pub/Sub Channels:**
```
Channel: bus:location:updates
Message Format: JSON
{
  "busId": "uuid",
  "routeId": "uuid",
  "routeName": "string",
  "latitude": 23.7104,
  "longitude": 90.4074,
  "timestamp": "2024-01-15T10:30:00Z",
  "driverName": "string"
}

Channel: route:updates
Message Format: JSON
{
  "action": "created|updated|deleted",
  "routeId": "uuid",
  "route": { /* full route object */ }
}
```

---

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

This system is primarily integration-heavy with significant infrastructure concerns (WebSocket management, database operations, Redis pub/sub). However, several core validation and business logic functions are suitable for property-based testing. The properties below focus on pure functions and deterministic logic that can be tested independently of external infrastructure.

### Property 1: JWT Generation with Role-Based Expiry

*For any* valid user credentials with a specified role (driver or admin), the generated JWT SHALL contain the correct role claim, have an expiry time matching the role-specific duration (12 hours for drivers, 8 hours for admins), and be verifiable with the signing secret.

**Validates: Requirements 3.1, 4.1**

### Property 2: Password Hashing Verification

*For any* password string, the bcrypt hash generated by the Auth Service SHALL be verifiable against the original password using bcrypt's verification function, and SHALL NOT be verifiable against any other password string.

**Validates: Requirements 3.4**

### Property 3: GPS Update Rate Limiting

*For any* sequence of GPS update timestamps from the same driver session, if two consecutive timestamps are less than 5 seconds apart, the second update SHALL be rejected with HTTP status 429.

**Validates: Requirements 2.2**

### Property 4: Authentication Rate Limiting

*For any* sequence of failed authentication attempts from the same IP address, if 5 or more failures occur within a 60-second window, all subsequent attempts within that window SHALL be rejected with HTTP status 429.

**Validates: Requirements 4.3**

### Property 5: GPS Reading Queue Management

*For any* sequence of GPS readings with timestamps, when queued locally during network outage, the queue SHALL maintain a maximum size of 500 readings by discarding the oldest reading when full, and SHALL discard all readings with timestamps older than 30 minutes before transmission.

**Validates: Requirements 2.5, 2.6**


### Property 6: Route Data Validation

*For any* route data with a name and list of stops, the validation logic SHALL accept the data if and only if: the name is 1-100 characters and unique, the stop list contains 2-50 stops, and each stop has a name (1-100 characters) and valid coordinates (latitude -90 to 90, longitude -180 to 180). Invalid data SHALL be rejected with HTTP status 422.

**Validates: Requirements 5.1, 5.2, 5.3**

### Property 7: GPS Coordinate Validation

*For any* GPS payload, the validation logic SHALL reject the payload with HTTP status 422 if: any required field (busId, latitude, longitude, timestamp) is missing, latitude is outside [-90, 90], longitude is outside [-180, 180], timestamp is malformed, or timestamp is older than 5 minutes relative to server time.

**Validates: Requirements 6.3, 6.4, 6.5**

### Property 8: Bus Marker Labeling

*For any* bus data object containing optional driverName and routeName fields, the marker label rendering function SHALL include the driverName if present or "Unassigned" if absent, and SHALL include the routeName if present or "No Route" if absent.

**Validates: Requirements 1.7, 7.4**

### Property 9: API Gateway Request Routing

*For any* HTTP request with a URL path, the API Gateway routing logic SHALL forward requests with paths beginning with `/routes` to the Route Service, paths beginning with `/auth` to the Auth Service, paths beginning with `/location` to the Location Service, and SHALL return HTTP status 404 for any path not matching these prefixes.

**Validates: Requirements 8.1, 8.5**

### Property 10: Gateway JWT Validation

*For any* JWT token in an Authorization header, the API Gateway SHALL reject the request with HTTP status 401 if the token is malformed, expired, or has an invalid signature. Additionally, if 10 or more failed JWT validation attempts occur from the same IP within a 60-second window, subsequent requests from that IP SHALL be rejected with HTTP status 429.

**Validates: Requirements 8.3**

### Property 11: CORS Origin Enforcement

*For any* HTTP request with an Origin header, the API Gateway SHALL accept the request only if the origin is in the configured allowed origins list (Admin Panel and Rider App origins), and SHALL reject requests from any other origin with HTTP status 403.

**Validates: Requirements 8.4**

---

## Error Handling

### Error Response Format

All services SHALL return errors in a consistent JSON format:

```json
{
  "error": {
    "code": "string (machine-readable error code)",
    "message": "string (human-readable description)",
    "details": "object (optional, additional context)",
    "timestamp": "string (ISO 8601)"
  }
}
```

### HTTP Status Code Usage

| Status Code | Usage |
|-------------|-------|
| 200 OK | Successful GET, PUT requests |
| 201 Created | Successful POST creating new resource |
| 204 No Content | Successful DELETE |
| 400 Bad Request | Malformed request syntax |
| 401 Unauthorized | Missing or invalid authentication |
| 403 Forbidden | Valid auth but insufficient permissions |
| 404 Not Found | Resource does not exist |
| 409 Conflict | Resource conflict (duplicate name, active bus on route) |
| 422 Unprocessable Entity | Validation error (invalid coordinates, missing fields) |
| 429 Too Many Requests | Rate limit exceeded |
| 500 Internal Server Error | Unexpected server error |
| 503 Service Unavailable | Service down or connection limit reached |

### Service-Specific Error Handling

**API Gateway:**
- Log all 401/403 errors with source IP and timestamp
- Implement exponential backoff for downstream service failures
- Return 503 if all instances of a downstream service are unavailable
- Gracefully handle WebSocket upgrade failures

**Auth Service:**
- Never expose whether username or password was incorrect (always return generic "Invalid credentials")
- Log all failed authentication attempts with IP, username, and timestamp
- Implement account lockout after repeated failures (future enhancement)
- Handle database connection failures gracefully

**Location Service:**
- Reject stale GPS coordinates (> 5 minutes old) to prevent outdated data
- Handle Redis connection failures by returning 503
- Gracefully close WebSocket connections on service shutdown
- Log all validation errors for monitoring

**Route Service:**
- Validate foreign key constraints before deletion
- Handle concurrent route updates with optimistic locking
- Rollback database transactions on any validation failure
- Publish route change notifications even if some subscribers are unavailable

### Client-Side Error Handling

**Rider App:**
- Display user-friendly error messages for all error codes
- Implement automatic retry with exponential backoff for 503 errors
- Cache last known bus positions for offline viewing
- Show persistent offline indicator when network is unavailable

**Driver App:**
- Queue GPS readings locally during network outages (max 500, 30-minute age limit)
- Prompt for re-authentication on 401 errors
- Display clear error messages for rate limiting (429)
- Implement background service restart on app crash

**Admin Panel:**
- Display validation errors inline on form fields
- Confirm destructive actions (route deletion) with modal dialogs
- Show loading states during async operations
- Implement optimistic UI updates with rollback on error

---

## Testing Strategy

### Testing Approach

The system requires a dual testing approach combining property-based tests for core validation logic with integration tests for infrastructure concerns.

**Property-Based Testing:**
- Focus on pure functions: validation logic, JWT generation, rate limiting, routing logic
- Use **gopter** (Go property-based testing library)
- Minimum 100 iterations per property test
- Each test tagged with: `Feature: university-bus-tracker, Property {number}: {property_text}`

**Integration Testing:**
- Focus on infrastructure: WebSocket connections, database operations, Redis pub/sub
- Use **testcontainers-go** for PostgreSQL and Redis
- Mock external dependencies where appropriate
- Test end-to-end flows with real infrastructure

**Unit Testing:**
- Focus on specific examples and edge cases
- Test error handling paths
- Test business logic with mocked dependencies

### Test Coverage by Service

**API Gateway:**
- **Property Tests:**
  - Request routing logic (Property 9)
  - JWT validation (Property 10)
  - CORS enforcement (Property 11)
- **Integration Tests:**
  - WebSocket proxying to Location Service
  - Downstream service failure handling
  - Rate limiting with Redis
- **Unit Tests:**
  - Middleware chain execution
  - Error response formatting

**Auth Service:**
- **Property Tests:**
  - JWT generation with role-based expiry (Property 1)
  - Password hashing verification (Property 2)
  - Authentication rate limiting (Property 4)
- **Integration Tests:**
  - Database connection and query execution
  - Failed login attempt tracking in Redis
- **Unit Tests:**
  - Specific credential validation scenarios
  - Token expiry edge cases

**Location Service:**
- **Property Tests:**
  - GPS update rate limiting (Property 3)
  - GPS coordinate validation (Property 7)
- **Integration Tests:**
  - WebSocket connection management (30 drivers, 300 riders)
  - Redis pub/sub message broadcasting
  - Last-known position caching with TTL
  - Connection limit enforcement
- **Unit Tests:**
  - WebSocket message formatting
  - Active bus tracking
  - Broadcast timeout handling

**Route Service:**
- **Property Tests:**
  - Route data validation (Property 6)
- **Integration Tests:**
  - Database CRUD operations
  - Route change notification via Redis pub/sub
  - Foreign key constraint enforcement
  - Concurrent update handling
- **Unit Tests:**
  - Stop ordering logic
  - Bus assignment validation


**Driver App:**
- **Property Tests:**
  - GPS reading queue management (Property 5)
  - Marker labeling (Property 8, rider portion)
- **Integration Tests:**
  - Background GPS broadcasting
  - WebSocket connection and reconnection
  - Local queue persistence
  - Network state detection
- **Unit Tests:**
  - Session lifecycle management
  - Offline indicator display
  - Retry logic (3 attempts, 5-second intervals)

**Rider App:**
- **Property Tests:**
  - Marker labeling (Property 8, rider portion)
- **Integration Tests:**
  - WebSocket connection and reconnection
  - Initial data fetching
  - Real-time map updates
- **Unit Tests:**
  - Offline indicator display
  - Error message and retry UI
  - Network state detection

**Admin Panel:**
- **Property Tests:**
  - Marker labeling (Property 8, admin portion)
- **Integration Tests:**
  - Route CRUD operations via API
  - WebSocket connection for live map
  - Form validation and submission
- **Unit Tests:**
  - Form field validation
  - Optimistic UI updates
  - Error display and rollback

### Load and Performance Testing

**Location Service Load Tests:**
- 30 concurrent driver connections sending GPS updates every 5 seconds
- 300 concurrent rider WebSocket connections receiving updates
- Measure: broadcast latency (target: < 2 seconds), memory usage, CPU usage
- Verify: connection rejection at limits (503 for HTTP, close code 1013 for WebSocket)

**API Gateway Load Tests:**
- 1000 requests/second across all endpoints
- Measure: routing latency, JWT validation overhead
- Verify: rate limiting enforcement

**Database Performance Tests:**
- Route queries with 50 stops
- Concurrent route updates
- Measure: query execution time, connection pool usage

### Property-Based Testing Configuration

**Library:** gopter (https://github.com/leanovate/gopter)

**Test Configuration:**
```go
parameters := gopter.DefaultTestParameters()
parameters.MinSuccessfulTests = 100
parameters.MaxSize = 1000
```

**Tag Format:**
```go
// Feature: university-bus-tracker, Property 1: JWT generation with role-based expiry
func TestJWTGenerationProperty(t *testing.T) {
    // property test implementation
}
```

### Continuous Integration

**GitHub Actions Workflow:**
1. Run unit tests on every commit
2. Run property tests (100 iterations) on every PR
3. Run integration tests with testcontainers on every PR
4. Run load tests on main branch weekly
5. Generate coverage report (target: 80% for business logic)

**Pre-deployment Checks:**
- All tests passing
- No critical security vulnerabilities (gosec)
- Code formatted (gofmt)
- Linting passed (golangci-lint)

---

## Deployment Strategy

### Infrastructure Overview

**Hosting Platform:** Railway (Free Tier)
- 500 hours/month execution time (shared across all services)
- 512 MB RAM per service
- 1 GB disk per service
- Automatic HTTPS and custom domains

**Database:** Neon PostgreSQL (Free Tier)
- 0.5 GB storage
- 1 GB RAM
- Automatic backups (7-day retention)
- Connection pooling via PgBouncer

**Cache/Pub-Sub:** Upstash Redis (Free Tier)
- 256 MB storage
- 10,000 commands/day
- TLS encryption
- Global replication (optional)

### Container Configuration

**Base Dockerfile (all services):**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

**Service-Specific Resource Limits:**

| Service | RAM Limit | CPU Limit | Replicas |
|---------|-----------|-----------|----------|
| API Gateway | 256 MB | 0.5 vCPU | 1 |
| Auth Service | 128 MB | 0.25 vCPU | 1 |
| Location Service | 256 MB | 0.5 vCPU | 1 |
| Route Service | 128 MB | 0.25 vCPU | 1 |

### Environment Variables

**API Gateway:**
```bash
PORT=8080
AUTH_SERVICE_URL=http://auth-service:8081
LOCATION_SERVICE_URL=http://location-service:8082
ROUTE_SERVICE_URL=http://route-service:8083
JWT_SECRET=${JWT_SECRET}
ALLOWED_ORIGINS=https://admin.example.com,https://rider.example.com
RATE_LIMIT_WINDOW=60s
RATE_LIMIT_MAX=10
```

**Auth Service:**
```bash
PORT=8081
DATABASE_URL=${NEON_DATABASE_URL}
JWT_SECRET=${JWT_SECRET}
DRIVER_TOKEN_EXPIRY=12h
ADMIN_TOKEN_EXPIRY=8h
BCRYPT_COST=12
```

**Location Service:**
```bash
PORT=8082
REDIS_URL=${UPSTASH_REDIS_URL}
MAX_DRIVER_CONNECTIONS=30
MAX_RIDER_CONNECTIONS=300
LOCATION_TTL=30m
BROADCAST_TIMEOUT=2s
MIN_UPDATE_INTERVAL=5s
MAX_TIMESTAMP_AGE=5m
```

**Route Service:**
```bash
PORT=8083
DATABASE_URL=${NEON_DATABASE_URL}
REDIS_URL=${UPSTASH_REDIS_URL}
```


### Deployment Process

**1. Initial Setup:**
```bash
# Create Railway project
railway init

# Add PostgreSQL (Neon)
railway add --plugin neon

# Add Redis (Upstash)
railway add --plugin upstash-redis

# Set secrets
railway variables set JWT_SECRET=$(openssl rand -base64 32)
```

**2. Service Deployment:**
```bash
# Deploy each service
cd api-gateway && railway up
cd auth-service && railway up
cd location-service && railway up
cd route-service && railway up
```

**3. Database Migration:**
```bash
# Run migrations using golang-migrate
migrate -path ./migrations -database ${DATABASE_URL} up
```

**4. Seed Initial Data:**
```bash
# Create admin account
curl -X POST https://api.example.com/auth/seed-admin \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"${ADMIN_PASSWORD}"}'

# Create driver accounts (pre-created by developer)
curl -X POST https://api.example.com/auth/seed-drivers \
  -H "Content-Type: application/json" \
  -d '{"drivers":[{"username":"driver1","password":"${DRIVER1_PASSWORD}"}]}'
```

### Health Checks

Each service SHALL expose a `/health` endpoint:

```go
type HealthResponse struct {
    Status      string            `json:"status"` // "healthy" | "degraded" | "unhealthy"
    Timestamp   string            `json:"timestamp"`
    Version     string            `json:"version"`
    Dependencies map[string]string `json:"dependencies"`
}
```

**Health Check Logic:**
- **Healthy:** All dependencies reachable, service operational
- **Degraded:** Some non-critical dependencies unavailable
- **Unhealthy:** Critical dependencies unavailable or service error

**Railway Health Check Configuration:**
```yaml
healthcheck:
  path: /health
  interval: 30s
  timeout: 5s
  retries: 3
```

### Monitoring and Logging

**Logging Strategy:**
- Structured JSON logs to stdout
- Log levels: DEBUG, INFO, WARN, ERROR
- Include: timestamp, service name, request ID, user ID (if authenticated)

**Log Aggregation:**
- Railway built-in log viewer
- Optional: Export to external service (Logtail, Papertrail)

**Metrics to Monitor:**
- Request rate and latency per endpoint
- WebSocket connection count (drivers and riders)
- GPS update rate per bus
- Redis memory usage
- PostgreSQL connection pool usage
- Error rate by status code

**Alerting Thresholds:**
- Error rate > 5% for 5 minutes
- WebSocket connections > 280 (93% of limit)
- Redis memory > 240 MB (93% of limit)
- Database storage > 450 MB (90% of limit)
- Service response time > 1 second (p95)

### Scaling Considerations

**Current Limits (Free Tier):**
- 30 concurrent buses
- 300 concurrent riders
- ~10,000 GPS updates/day (Redis command limit)

**Future Scaling Path (if needed):**
1. Upgrade to Railway Pro ($5/month per service)
2. Upgrade Upstash Redis to paid tier ($10/month for 1 GB)
3. Upgrade Neon to paid tier ($19/month for 10 GB)
4. Implement horizontal scaling for Location Service (multiple replicas with Redis pub/sub)
5. Add CDN for static assets (Cloudflare free tier)

### Disaster Recovery

**Backup Strategy:**
- PostgreSQL: Automatic daily backups by Neon (7-day retention)
- Redis: No persistent backups (cache only, can be rebuilt)
- Application state: Stateless services, no backup needed

**Recovery Procedures:**
1. **Database corruption:** Restore from Neon backup (< 1 hour)
2. **Service failure:** Railway auto-restart (< 1 minute)
3. **Redis failure:** Service continues with degraded performance, Redis auto-restart
4. **Complete outage:** Redeploy all services from Git (< 10 minutes)

**Data Loss Scenarios:**
- GPS location data: Max 30 minutes loss (Redis TTL)
- Route/stop data: No loss (PostgreSQL with backups)
- User credentials: No loss (PostgreSQL with backups)

---

## Security Considerations

### Authentication and Authorization

**JWT Security:**
- Use HS256 algorithm with 256-bit secret
- Rotate JWT secret every 90 days
- Include role claim for RBAC
- Set appropriate expiry (12h drivers, 8h admins)
- Validate signature, expiry, and claims on every request

**Password Security:**
- Bcrypt with cost factor 12
- Minimum password length: 8 characters
- Require password change on first login (future enhancement)
- Never log passwords or password hashes

**Rate Limiting:**
- Failed authentication: 5 attempts per IP per 60 seconds
- Failed JWT validation: 10 attempts per IP per 60 seconds
- GPS updates: 1 per 5 seconds per driver session
- Implement IP-based rate limiting with Redis

### Network Security

**HTTPS/TLS:**
- All external communication over HTTPS (Railway provides automatic TLS)
- WebSocket connections over WSS (secure WebSocket)
- Database connections over TLS (Neon enforces TLS)
- Redis connections over TLS (Upstash enforces TLS)

**CORS Configuration:**
- Whitelist only known origins (Admin Panel, Rider App)
- Allow credentials for authenticated requests
- Restrict allowed methods: GET, POST, PUT, DELETE, OPTIONS
- Set appropriate preflight cache duration

**API Security Headers:**
```go
// Add to all responses
headers := map[string]string{
    "X-Content-Type-Options": "nosniff",
    "X-Frame-Options": "DENY",
    "X-XSS-Protection": "1; mode=block",
    "Strict-Transport-Security": "max-age=31536000; includeSubDomains",
    "Content-Security-Policy": "default-src 'self'",
}
```

### Input Validation

**Validation Rules:**
- Validate all inputs against expected types and ranges
- Reject requests with unexpected fields
- Sanitize string inputs to prevent injection attacks
- Use parameterized queries for all database operations
- Validate GPS coordinates against realistic bounds for Bangladesh

**SQL Injection Prevention:**
- Use prepared statements exclusively
- Never concatenate user input into SQL queries
- Use ORM (GORM) with proper escaping

**XSS Prevention:**
- Escape all user-generated content in responses
- Set Content-Type headers correctly
- Use Content-Security-Policy headers

### Data Privacy

**Personal Data Handling:**
- Driver names: Visible only to admins
- User credentials: Hashed and never exposed
- GPS coordinates: Retained for 30 minutes only
- No tracking of individual riders

**Data Retention:**
- GPS locations: 30 minutes (Redis TTL)
- Route/stop data: Indefinite (until deleted by admin)
- User accounts: Indefinite (until deleted by developer)
- Logs: 7 days (Railway default)

**GDPR Considerations (if applicable):**
- Right to access: Provide user data export endpoint
- Right to deletion: Implement account deletion endpoint
- Data minimization: Collect only necessary data
- Purpose limitation: Use data only for tracking purposes

### Secrets Management

**Secret Storage:**
- Store secrets in Railway environment variables
- Never commit secrets to Git
- Use different secrets for dev/staging/production
- Rotate secrets regularly (90-day cycle)

**Required Secrets:**
- `JWT_SECRET`: 256-bit random string
- `DATABASE_URL`: Neon connection string
- `REDIS_URL`: Upstash connection string
- `ADMIN_PASSWORD`: Initial admin password
- `DRIVER_PASSWORDS`: Initial driver passwords

### Vulnerability Management

**Dependency Scanning:**
- Run `go mod tidy` regularly
- Use `govulncheck` to scan for known vulnerabilities
- Update dependencies monthly
- Monitor GitHub security advisories

**Security Auditing:**
- Run `gosec` static analysis on every commit
- Review authentication/authorization logic quarterly
- Conduct penetration testing before production launch
- Monitor Railway security bulletins

**Incident Response:**
1. Detect: Monitor logs for suspicious activity
2. Contain: Disable compromised accounts, rotate secrets
3. Investigate: Review logs, identify attack vector
4. Remediate: Patch vulnerabilities, restore from backup if needed
5. Document: Record incident details and lessons learned

---

## Implementation Details

### Technology Choices and Rationale

**Go + Gin Framework:**
- **Why Go:** Excellent concurrency support (goroutines) for WebSocket connections, low memory footprint, fast compilation
- **Why Gin:** Lightweight, fast HTTP router, middleware support, good documentation
- **Alternatives considered:** Node.js (higher memory usage), Python (slower performance)

**PostgreSQL on Neon:**
- **Why PostgreSQL:** ACID compliance, strong data integrity, excellent geospatial support (PostGIS if needed)
- **Why Neon:** Generous free tier, automatic backups, serverless architecture, connection pooling
- **Alternatives considered:** MySQL (less feature-rich), MongoDB (overkill for structured data)

**Redis on Upstash:**
- **Why Redis:** Fast in-memory operations, built-in pub/sub, TTL support, simple data structures
- **Why Upstash:** Generous free tier, TLS by default, REST API fallback, global replication
- **Alternatives considered:** In-memory Go maps (no persistence), RabbitMQ (more complex)

**Railway Deployment:**
- **Why Railway:** Simple deployment, automatic HTTPS, generous free tier, good DX
- **Alternatives considered:** Heroku (expensive), Fly.io (more complex), AWS (overkill)

### Key Go Packages

**Core Dependencies:**
```go
require (
    github.com/gin-gonic/gin v1.9.1           // HTTP framework
    github.com/golang-jwt/jwt/v5 v5.2.0       // JWT handling
    github.com/gorilla/websocket v1.5.1       // WebSocket support
    github.com/redis/go-redis/v9 v9.4.0       // Redis client
    gorm.io/gorm v1.25.5                      // ORM
    gorm.io/driver/postgres v1.5.4            // PostgreSQL driver
    golang.org/x/crypto v0.17.0               // Bcrypt hashing
    github.com/google/uuid v1.5.0             // UUID generation
    github.com/leanovate/gopter v0.2.9        // Property-based testing
)
```

### Project Structure

```
university-bus-tracker/
├── api-gateway/
│   ├── main.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── ratelimit.go
│   │   └── logging.go
│   ├── router/
│   │   └── router.go
│   └── Dockerfile
├── auth-service/
│   ├── main.go
│   ├── handlers/
│   │   └── auth.go
│   ├── models/
│   │   └── user.go
│   ├── services/
│   │   ├── jwt.go
│   │   └── password.go
│   └── Dockerfile
├── location-service/
│   ├── main.go
│   ├── handlers/
│   │   ├── location.go
│   │   └── websocket.go
│   ├── services/
│   │   ├── broadcast.go
│   │   ├── validation.go
│   │   └── ratelimit.go
│   ├── models/
│   │   └── location.go
│   └── Dockerfile
├── route-service/
│   ├── main.go
│   ├── handlers/
│   │   └── routes.go
│   ├── models/
│   │   ├── route.go
│   │   └── stop.go
│   ├── services/
│   │   ├── validation.go
│   │   └── pubsub.go
│   └── Dockerfile
├── shared/
│   ├── config/
│   │   └── config.go
│   ├── errors/
│   │   └── errors.go
│   └── logger/
│       └── logger.go
├── migrations/
│   ├── 001_create_users.up.sql
│   ├── 002_create_routes.up.sql
│   ├── 003_create_stops.up.sql
│   ├── 004_create_buses.up.sql
│   └── 005_create_route_assignments.up.sql
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```


### WebSocket Connection Management

**Connection Lifecycle:**

```go
type ConnectionManager struct {
    driverConns map[string]*websocket.Conn
    riderConns  map[string]*websocket.Conn
    mu          sync.RWMutex
    maxDrivers  int
    maxRiders   int
}

func (cm *ConnectionManager) AddDriver(id string, conn *websocket.Conn) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    if len(cm.driverConns) >= cm.maxDrivers {
        return errors.New("driver connection limit reached")
    }
    
    cm.driverConns[id] = conn
    return nil
}

func (cm *ConnectionManager) BroadcastToRiders(msg []byte) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    for _, conn := range cm.riderConns {
        go func(c *websocket.Conn) {
            c.SetWriteDeadline(time.Now().Add(2 * time.Second))
            c.WriteMessage(websocket.TextMessage, msg)
        }(conn)
    }
}
```

**Heartbeat Mechanism:**
- Send ping every 30 seconds
- Expect pong within 10 seconds
- Close connection if no pong received

**Graceful Shutdown:**
- Send close frame to all connections
- Wait up to 5 seconds for acknowledgment
- Force close remaining connections

### GPS Coordinate Processing Pipeline

```
Driver App → API Gateway → Location Service
                              ↓
                         Validate Payload
                              ↓
                         Check Rate Limit
                              ↓
                         Store in Redis (TTL: 30m)
                              ↓
                         Publish to Redis Channel
                              ↓
                         Broadcast to WebSocket Clients
```

**Validation Steps:**
1. Check required fields (busId, lat, lng, timestamp)
2. Validate coordinate ranges
3. Validate timestamp format and age
4. Check rate limit (5-second minimum interval)
5. Verify bus exists and is assigned to a route

**Broadcast Optimization:**
- Use goroutines for parallel WebSocket writes
- Set write deadline (2 seconds)
- Skip slow clients (don't block fast clients)
- Log broadcast failures for monitoring

### Database Query Optimization

**Indexes:**
- `users(username)` - Fast login lookups
- `routes(name)` - Unique constraint enforcement
- `stops(route_id, stop_order)` - Ordered stop retrieval
- `route_assignments(route_id)` - Fast assignment lookups
- `route_assignments(bus_id)` - Prevent duplicate assignments

**Query Patterns:**
```sql
-- Fetch all routes with stops (single query with JOIN)
SELECT r.id, r.name, s.id, s.name, s.latitude, s.longitude, s.stop_order
FROM routes r
LEFT JOIN stops s ON r.id = s.route_id
ORDER BY r.name, s.stop_order;

-- Check if route has active bus (before deletion)
SELECT COUNT(*) FROM route_assignments WHERE route_id = $1;

-- Get bus location with route info (Redis query)
HGETALL bus:location:{busId}
```

**Connection Pooling:**
```go
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
sqlDB, err := db.DB()
sqlDB.SetMaxOpenConns(10)
sqlDB.SetMaxIdleConns(5)
sqlDB.SetConnMaxLifetime(time.Hour)
```

---

## Future Enhancements

### Phase 2: AI-Based ETA Prediction

**Objective:** Provide estimated arrival times at each stop based on historical data and current traffic conditions.

**Implementation:**
- Collect historical GPS data (speed, route, time of day)
- Train ML model (e.g., XGBoost) on historical journey times
- Integrate with Google Maps Traffic API for real-time traffic data
- Store predictions in Redis with 5-minute TTL
- Display ETA on rider app for each stop

**Estimated Cost:** $10-20/month (Google Maps API)

### Phase 3: Push Notifications

**Objective:** Notify riders when their bus is approaching their stop.

**Implementation:**
- Integrate Firebase Cloud Messaging (FCM) for Android
- Allow riders to subscribe to specific routes/stops
- Calculate distance between bus and stop
- Send notification when bus is within 500m of subscribed stop
- Implement notification preferences (enable/disable, quiet hours)

**Estimated Cost:** Free (FCM free tier: unlimited notifications)

### Phase 4: Route Optimization

**Objective:** Suggest optimal routes based on rider demand and traffic patterns.

**Implementation:**
- Collect rider location data (opt-in)
- Analyze demand patterns by time and location
- Use routing algorithms (Dijkstra, A*) to suggest efficient routes
- Provide admin dashboard with route optimization recommendations
- A/B test new routes before permanent adoption

**Estimated Cost:** $5-10/month (additional compute)

### Phase 5: Driver Performance Analytics

**Objective:** Track driver performance metrics for management insights.

**Implementation:**
- Calculate metrics: on-time percentage, route completion rate, average speed
- Detect anomalies: excessive speeding, route deviations, long stops
- Generate weekly reports for transportation department
- Implement driver leaderboard (gamification)
- Privacy-preserving analytics (aggregate data only)

**Estimated Cost:** $0 (within existing infrastructure)

### Phase 6: Multi-University Support

**Objective:** Expand system to support multiple universities.

**Implementation:**
- Add `university_id` to all tables
- Implement tenant isolation in database queries
- Create super-admin role for cross-university management
- Support custom branding per university
- Implement usage-based pricing model

**Estimated Cost:** Variable (depends on scale)

### Phase 7: Offline-First Mobile Apps

**Objective:** Improve app usability in low-connectivity areas.

**Implementation:**
- Implement local database (SQLite) in mobile apps
- Sync route/stop data when online
- Cache map tiles for offline viewing
- Queue all mutations (GPS updates, route changes) for sync
- Implement conflict resolution for concurrent updates

**Estimated Cost:** $0 (client-side only)

### Phase 8: Accessibility Features

**Objective:** Make the system accessible to users with disabilities.

**Implementation:**
- Voice announcements for bus arrivals (text-to-speech)
- High-contrast mode for visually impaired users
- Screen reader support (ARIA labels)
- Haptic feedback for important events
- Multilingual support (Bengali, English)

**Estimated Cost:** $0 (development time only)

---

## Appendix

### API Request/Response Examples

**Example 1: Driver Login**

Request:
```http
POST /auth/login HTTP/1.1
Host: api.example.com
Content-Type: application/json

{
  "username": "driver1",
  "password": "securepassword123",
  "role": "driver"
}
```

Response:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2024-01-16T10:30:00Z",
  "role": "driver",
  "userId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Example 2: GPS Update**

Request:
```http
POST /location/update HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "busId": "660e8400-e29b-41d4-a716-446655440001",
  "latitude": 23.7104,
  "longitude": 90.4074,
  "timestamp": "2024-01-15T10:30:00Z",
  "accuracy": 10.5,
  "speed": 8.3
}
```

Response:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "accepted",
  "broadcastedTo": 45
}
```

**Example 3: Create Route**

Request:
```http
POST /routes HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "name": "Route A - Science Building Loop",
  "stops": [
    {
      "name": "Main Gate",
      "latitude": 23.7104,
      "longitude": 90.4074
    },
    {
      "name": "Science Building",
      "latitude": 23.7115,
      "longitude": 90.4085
    },
    {
      "name": "Library",
      "latitude": 23.7125,
      "longitude": 90.4095
    }
  ]
}
```

Response:
```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "name": "Route A - Science Building Loop",
  "stops": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440003",
      "name": "Main Gate",
      "latitude": 23.7104,
      "longitude": 90.4074,
      "order": 0
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440004",
      "name": "Science Building",
      "latitude": 23.7115,
      "longitude": 90.4085,
      "order": 1
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440005",
      "name": "Library",
      "latitude": 23.7125,
      "longitude": 90.4095,
      "order": 2
    }
  ],
  "assignedBusId": null,
  "assignedDriverName": null
}
```

**Example 4: WebSocket Location Update**

Server → Client:
```json
{
  "type": "location_update",
  "busId": "660e8400-e29b-41d4-a716-446655440001",
  "routeId": "770e8400-e29b-41d4-a716-446655440002",
  "routeName": "Route A - Science Building Loop",
  "latitude": 23.7104,
  "longitude": 90.4074,
  "timestamp": "2024-01-15T10:30:00Z",
  "driverName": "John Doe"
}
```

### Database Migration Scripts

**001_create_users.up.sql:**
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('driver', 'admin')),
    full_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);
```

**002_create_routes.up.sql:**
```sql
CREATE TABLE routes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_routes_name ON routes(name);
```

### Configuration File Examples

**config.yaml (development):**
```yaml
api_gateway:
  port: 8080
  auth_service_url: http://localhost:8081
  location_service_url: http://localhost:8082
  route_service_url: http://localhost:8083
  allowed_origins:
    - http://localhost:3000
    - http://localhost:3001
  jwt_secret: dev-secret-change-in-production
  rate_limit_window: 60s
  rate_limit_max: 10

auth_service:
  port: 8081
  database_url: postgres://user:pass@localhost:5432/bustrack_dev
  jwt_secret: dev-secret-change-in-production
  driver_token_expiry: 12h
  admin_token_expiry: 8h
  bcrypt_cost: 10

location_service:
  port: 8082
  redis_url: redis://localhost:6379
  max_driver_connections: 30
  max_rider_connections: 300
  location_ttl: 30m
  broadcast_timeout: 2s
  min_update_interval: 5s
  max_timestamp_age: 5m

route_service:
  port: 8083
  database_url: postgres://user:pass@localhost:5432/bustrack_dev
  redis_url: redis://localhost:6379
```

### Glossary of Technical Terms

- **Bcrypt:** Password hashing algorithm designed to be slow and resistant to brute-force attacks
- **CORS:** Cross-Origin Resource Sharing, security feature that restricts web pages from making requests to different domains
- **Goroutine:** Lightweight thread managed by the Go runtime
- **JWT:** JSON Web Token, compact URL-safe means of representing claims between two parties
- **Middleware:** Software that sits between the client and server, processing requests/responses
- **Pub/Sub:** Publish-Subscribe messaging pattern where publishers send messages to channels and subscribers receive them
- **Rate Limiting:** Technique to control the rate of requests a client can make to prevent abuse
- **TTL:** Time To Live, duration after which data expires and is automatically deleted
- **WebSocket:** Protocol providing full-duplex communication channels over a single TCP connection
- **GORM:** Go Object-Relational Mapping library for database operations
- **PgBouncer:** Lightweight connection pooler for PostgreSQL

---

## Document Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2024-01-15 | System | Initial design document created |

---

**End of Design Document**
