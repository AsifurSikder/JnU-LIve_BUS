# Implementation Plan: University Bus Tracker

## Overview

This implementation plan breaks down the University Bus Tracker system into discrete coding tasks. The system consists of four Go microservices (API Gateway, Auth Service, Location Service, Route Service), two Android applications (Rider App, Driver App), and a React web application (Admin Panel). The implementation follows a bottom-up approach, starting with core infrastructure and data models, then building service-specific functionality, and finally integrating client applications.

The plan prioritizes early validation through property-based tests for core validation logic, complemented by integration tests for infrastructure concerns. Each task references specific requirements to ensure traceability.

## Tasks

- [x] 1. Set up project structure and shared infrastructure
  - [x] 1.1 Initialize Go project structure for all microservices
    - Create directory structure: `api-gateway/`, `auth-service/`, `location-service/`, `route-service/`
    - Initialize Go modules for each service with `go mod init`
    - Set up shared configuration package for common types and utilities
    - Create Dockerfiles for each service using multi-stage builds
    - Set up `.env.example` files with required environment variables
    - _Requirements: 8.1, 9.5_
  
  - [x] 1.2 Set up PostgreSQL schema and migrations
    - Create database migration files for `users`, `routes`, `stops`, `buses`, and `route_assignments` tables
    - Implement migration runner using `golang-migrate` or similar
    - Add indexes as specified in design document
    - Create seed data script for initial driver and admin accounts
    - _Requirements: 3.4, 4.1, 5.1_
  
  - [x] 1.3 Set up Redis connection and pub/sub infrastructure
    - Implement Redis client wrapper with connection pooling
    - Create pub/sub manager for `bus:location:updates` and `route:updates` channels
    - Implement Redis key helpers for consistent key naming
    - Add Redis health check endpoint
    - _Requirements: 6.1, 6.2, 5.9_
  
  - [x] 1.4 Implement shared JWT utilities
    - Create JWT generation function with role-based expiry
    - Create JWT validation and parsing function
    - Implement JWT claims structure with `sub`, `role`, `exp`, `iat` fields
    - Add JWT secret configuration loading
    - _Requirements: 3.1, 4.1_


  - [-] 1.5 Write property test for JWT generation with role-based expiry
    - **Property 1: JWT Generation with Role-Based Expiry**
    - **Validates: Requirements 3.1, 4.1**
    - Generate JWTs for random user credentials with driver/admin roles
    - Verify correct role claim, expiry time (12h for drivers, 8h for admins), and signature verification
    - Use gopter with 100 iterations

- [ ] 2. Implement Auth Service
  - [~] 2.1 Create Auth Service HTTP server with Gin
    - Set up Gin router with `/auth/login` endpoint
    - Implement configuration loading from environment variables
    - Add database connection with connection pooling
    - Implement health check endpoint
    - _Requirements: 3.1, 4.1_
  
  - [~] 2.2 Implement password hashing and verification
    - Create bcrypt password hashing function with configurable cost (default 12)
    - Create password verification function
    - _Requirements: 3.4_
  
  - [~] 2.3 Write property test for password hashing verification
    - **Property 2: Password Hashing Verification**
    - **Validates: Requirements 3.4**
    - Generate random passwords, hash them, verify against original and different passwords
    - Use gopter with 100 iterations
  
  - [~] 2.4 Implement login endpoint with credential validation
    - Parse and validate login request body (username, password, role)
    - Query database for user by username and role
    - Verify password using bcrypt
    - Generate JWT on successful authentication
    - Return appropriate error responses (401, 422, 503)
    - _Requirements: 3.1, 3.2, 3.3, 4.1, 4.2_
  
  - [~] 2.5 Implement authentication rate limiting
    - Create Redis-based rate limiter for failed login attempts
    - Track failed attempts by source IP with 60-second TTL
    - Reject requests with 429 after 5 failures (admin) or 10 failures (driver)
    - _Requirements: 4.3_


  - [~] 2.6 Write property test for authentication rate limiting
    - **Property 4: Authentication Rate Limiting**
    - **Validates: Requirements 4.3**
    - Generate sequences of failed auth attempts from same IP
    - Verify 429 response after threshold (5 for admin, 10 for driver) within 60-second window
    - Use gopter with 100 iterations
  
  - [~] 2.7 Write integration tests for Auth Service
    - Test successful driver and admin login with database
    - Test invalid credentials handling
    - Test rate limiting with Redis
    - Test JWT expiry edge cases
    - _Requirements: 3.1, 3.2, 3.3, 4.1, 4.2, 4.3_

- [~] 3. Checkpoint - Auth Service validation
  - Ensure all tests pass, verify Auth Service can authenticate users and issue JWTs. Ask the user if questions arise.

- [ ] 4. Implement Route Service
  - [~] 4.1 Create Route Service HTTP server with Gin
    - Set up Gin router with `/routes` endpoints (GET, POST, PUT, DELETE)
    - Implement configuration loading from environment variables
    - Add database and Redis connections
    - Implement health check endpoint
    - _Requirements: 5.1, 5.2, 5.4, 5.5_
  
  - [~] 4.2 Implement route data validation logic
    - Create validation function for route name (1-100 chars, unique check)
    - Create validation function for stop list (2-50 stops)
    - Create validation function for stop data (name 1-100 chars, coordinates in range)
    - Return 422 for validation errors
    - _Requirements: 5.1, 5.2, 5.3_
  
  - [~] 4.3 Write property test for route data validation
    - **Property 6: Route Data Validation**
    - **Validates: Requirements 5.1, 5.2, 5.3**
    - Generate random route data with varying validity
    - Verify acceptance only for valid data (name 1-100 chars, 2-50 stops, valid coordinates)
    - Use gopter with 100 iterations


  - [~] 4.4 Implement GET /routes endpoint
    - Query database for all routes with joins to stops and route_assignments
    - Order stops by `stop_order` field
    - Include assigned bus and driver information
    - Return JSON response with route list
    - _Requirements: 5.1_
  
  - [~] 4.5 Implement POST /routes endpoint
    - Parse and validate request body
    - Check for duplicate route name (return 409 if exists)
    - Begin database transaction
    - Insert route record and stop records with ordering
    - Commit transaction or rollback on error
    - Publish route creation event to Redis `route:updates` channel
    - Return 201 with created route
    - _Requirements: 5.1, 5.7, 5.9_
  
  - [~] 4.6 Implement PUT /routes/:id endpoint
    - Parse and validate request body
    - Check if route exists (return 404 if not)
    - Check for duplicate route name if name changed (return 409)
    - Begin database transaction
    - Update route record and replace stops
    - Commit transaction or rollback on error
    - Publish route update event to Redis `route:updates` channel
    - Return 200 with updated route
    - _Requirements: 5.2, 5.3, 5.4, 5.9_
  
  - [~] 4.7 Implement DELETE /routes/:id endpoint
    - Check if route exists (return 404 if not)
    - Check if route has active bus assigned (return 409 if yes)
    - Delete route (cascade deletes stops via foreign key)
    - Publish route deletion event to Redis `route:updates` channel
    - Return 204
    - _Requirements: 5.5, 5.6, 5.9_
  
  - [~] 4.8 Implement POST /routes/:id/assign endpoint
    - Parse and validate request body (busId, driverId)
    - Verify route, bus, and driver exist (return 404 if not)
    - Upsert route_assignments record (replace existing assignment)
    - Return 200 with assignment details
    - _Requirements: 5.8_


  - [~] 4.9 Write integration tests for Route Service
    - Test route CRUD operations with database
    - Test duplicate name handling (409)
    - Test active bus deletion prevention (409)
    - Test route change notifications via Redis pub/sub
    - Test concurrent route updates
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9_

- [~] 5. Checkpoint - Route Service validation
  - Ensure all tests pass, verify Route Service can manage routes and stops. Ask the user if questions arise.

- [ ] 6. Implement Location Service core functionality
  - [~] 6.1 Create Location Service HTTP server with Gin
    - Set up Gin router with `/location/update` endpoint
    - Implement configuration loading from environment variables
    - Add Redis connection for caching and pub/sub
    - Implement health check endpoint
    - _Requirements: 6.1, 6.2_
  
  - [~] 6.2 Implement GPS coordinate validation logic
    - Create validation function for required fields (busId, latitude, longitude, timestamp)
    - Create validation function for coordinate ranges (lat -90 to 90, lon -180 to 180)
    - Create validation function for timestamp format and age (< 5 minutes)
    - Return 422 for validation errors
    - _Requirements: 6.3, 6.4, 6.5_
  
  - [~] 6.3 Write property test for GPS coordinate validation
    - **Property 7: GPS Coordinate Validation**
    - **Validates: Requirements 6.3, 6.4, 6.5**
    - Generate random GPS payloads with varying validity
    - Verify rejection for missing fields, out-of-range coordinates, malformed/stale timestamps
    - Use gopter with 100 iterations
  
  - [~] 6.4 Implement GPS update rate limiting per driver session
    - Create Redis-based rate limiter tracking last update timestamp per driver
    - Check if last update was < 5 seconds ago
    - Return 429 if rate limit exceeded
    - Update timestamp on successful update
    - _Requirements: 2.2_


  - [~] 6.5 Write property test for GPS update rate limiting
    - **Property 3: GPS Update Rate Limiting**
    - **Validates: Requirements 2.2**
    - Generate sequences of GPS update timestamps from same driver session
    - Verify 429 rejection when consecutive timestamps < 5 seconds apart
    - Use gopter with 100 iterations
  
  - [~] 6.6 Implement POST /location/update endpoint
    - Parse and validate request body
    - Verify JWT and extract driver/bus information
    - Validate GPS coordinates and timestamp
    - Check rate limit
    - Store location in Redis hash with 30-minute TTL
    - Add bus to active buses set
    - Publish location update to Redis `bus:location:updates` channel
    - Return 200 with broadcast count
    - _Requirements: 2.1, 2.2, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6_
  
  - [~] 6.7 Write integration tests for Location Service GPS updates
    - Test GPS coordinate storage in Redis with TTL
    - Test rate limiting with Redis
    - Test location broadcast via Redis pub/sub
    - Test active bus tracking
    - _Requirements: 2.1, 2.2, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6_

- [ ] 7. Implement Location Service WebSocket functionality
  - [~] 7.1 Set up WebSocket server with gorilla/websocket
    - Create WebSocket upgrader with CORS configuration
    - Implement WebSocket connection handler
    - Create connection manager tracking active connections
    - Implement connection limit enforcement (300 concurrent)
    - _Requirements: 1.3, 7.2, 9.2, 9.3_
  
  - [~] 7.2 Implement WebSocket message broadcasting
    - Subscribe to Redis `bus:location:updates` channel
    - Parse location update messages
    - Broadcast to all connected WebSocket clients within 2 seconds
    - Handle client disconnections gracefully
    - Track broadcast count for metrics
    - _Requirements: 1.2, 6.2, 7.3_


  - [~] 7.3 Implement WebSocket connection lifecycle management
    - Handle WebSocket upgrade requests at `/ws/location`
    - Add new connections to connection manager
    - Send initial bus positions on connection
    - Remove connections on disconnect or error
    - Reject new connections with 503 when limit reached
    - _Requirements: 1.1, 1.6, 7.1, 7.2, 7.5, 9.2, 9.3_
  
  - [~] 7.4 Write integration tests for WebSocket functionality
    - Test WebSocket connection and message reception
    - Test connection limit enforcement (300 concurrent)
    - Test broadcast latency (< 2 seconds)
    - Test graceful disconnection handling
    - Test reconnection scenarios
    - _Requirements: 1.2, 1.3, 1.5, 1.6, 7.2, 7.3, 7.5, 9.2, 9.3_

- [~] 8. Checkpoint - Location Service validation
  - Ensure all tests pass, verify Location Service can process GPS updates and broadcast via WebSocket. Ask the user if questions arise.

- [ ] 9. Implement API Gateway
  - [~] 9.1 Create API Gateway HTTP server with Gin
    - Set up Gin router with middleware chain
    - Implement configuration loading for downstream service URLs
    - Add health check endpoint aggregating downstream services
    - _Requirements: 8.1_
  
  - [~] 9.2 Implement request routing middleware
    - Create routing logic for `/auth/*`, `/routes/*`, `/location/*` paths
    - Forward requests to appropriate downstream services
    - Return 404 for unmatched paths
    - Handle downstream service errors with appropriate status codes
    - _Requirements: 8.1, 8.5_
  
  - [~] 9.3 Write property test for API Gateway request routing
    - **Property 9: API Gateway Request Routing**
    - **Validates: Requirements 8.1, 8.5**
    - Generate random HTTP request paths
    - Verify correct routing to services or 404 for unmatched paths
    - Use gopter with 100 iterations


  - [~] 9.4 Implement JWT validation middleware
    - Extract JWT from Authorization header
    - Validate JWT signature, expiry, and format
    - Reject invalid/expired tokens with 401
    - Attach user claims to request context for downstream use
    - _Requirements: 2.7, 8.3_
  
  - [~] 9.5 Write property test for Gateway JWT validation
    - **Property 10: Gateway JWT Validation**
    - **Validates: Requirements 8.3**
    - Generate random JWTs (valid, malformed, expired, invalid signature)
    - Verify 401 rejection for invalid tokens
    - Use gopter with 100 iterations
  
  - [~] 9.6 Implement JWT validation rate limiting
    - Track failed JWT validation attempts by source IP in Redis
    - Reject with 429 after 10 failures within 60-second window
    - Log failed attempts with IP and timestamp
    - _Requirements: 8.3_
  
  - [~] 9.7 Implement CORS middleware
    - Parse Origin header from requests
    - Check against configured allowed origins list
    - Reject requests from unauthorized origins with 403
    - Add appropriate CORS headers for allowed origins
    - Support preflight OPTIONS requests
    - _Requirements: 8.4_
  
  - [~] 9.8 Write property test for CORS origin enforcement
    - **Property 11: CORS Origin Enforcement**
    - **Validates: Requirements 8.4**
    - Generate random Origin headers
    - Verify acceptance only for configured allowed origins, 403 for others
    - Use gopter with 100 iterations
  
  - [~] 9.9 Implement role-based access control middleware
    - Check JWT role claim against endpoint requirements
    - Reject requests without required role with 403
    - Allow admin-only endpoints only for admin role
    - Allow driver-only endpoints only for driver role
    - _Requirements: 4.4, 5.1, 5.2, 5.4, 5.5, 5.8_


  - [~] 9.10 Implement WebSocket proxy to Location Service
    - Detect WebSocket upgrade requests (Connection: Upgrade, Upgrade: websocket)
    - Proxy WebSocket connections to Location Service
    - Handle upgrade failures gracefully
    - Maintain bidirectional message forwarding
    - _Requirements: 8.2_
  
  - [~] 9.11 Write integration tests for API Gateway
    - Test request routing to all downstream services
    - Test JWT validation and rate limiting
    - Test CORS enforcement
    - Test role-based access control
    - Test WebSocket proxying
    - Test downstream service failure handling
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [~] 10. Checkpoint - API Gateway validation
  - Ensure all tests pass, verify API Gateway correctly routes requests and enforces security. Ask the user if questions arise.

- [ ] 11. Implement Driver App (Android)
  - [~] 11.1 Set up Android project structure
    - Create Android project with Kotlin
    - Add dependencies: Retrofit, OkHttp, Scarlet (WebSocket), Google Play Services Location
    - Set up MVVM architecture with ViewModel and LiveData
    - Configure permissions: ACCESS_FINE_LOCATION, ACCESS_BACKGROUND_LOCATION, INTERNET
    - _Requirements: 2.1, 2.3_
  
  - [~] 11.2 Implement login screen and authentication
    - Create login UI with username and password fields
    - Implement login API call to `/auth/login` with driver role
    - Store JWT in encrypted SharedPreferences
    - Handle authentication errors (401, 422, 503)
    - Navigate to route selection on successful login
    - _Requirements: 2.7, 3.1, 3.2, 3.3, 3.5_
  
  - [~] 11.3 Implement route selection screen
    - Fetch available routes from `/routes` endpoint
    - Display route list with names and stop counts
    - Allow driver to select a route to start session
    - Store selected route information locally
    - _Requirements: 2.1_


  - [~] 11.4 Implement GPS location tracking service
    - Create foreground service for continuous GPS tracking
    - Request location updates from FusedLocationProviderClient
    - Set location update interval to 5 seconds
    - Handle location permission requests
    - Display persistent notification while service is active
    - _Requirements: 2.1, 2.3_
  
  - [~] 11.5 Implement GPS data transmission to backend
    - Send GPS coordinates to `/location/update` endpoint every 5 seconds
    - Include JWT in Authorization header
    - Format payload with busId, latitude, longitude, timestamp, accuracy, speed
    - Handle rate limiting (429) by adjusting send interval
    - Handle authentication errors (401) by stopping service and showing login screen
    - _Requirements: 2.1, 2.2, 2.7_
  
  - [~] 11.6 Implement offline GPS reading queue
    - Detect network connectivity changes
    - Queue GPS readings locally when offline (max 500 readings)
    - Store readings with timestamps in local database (Room)
    - Discard oldest reading when queue reaches 500
    - Transmit queued readings in order when connectivity restored
    - Discard readings older than 30 minutes before transmission
    - _Requirements: 2.5, 2.6_
  
  - [~] 11.7 Write property test for GPS reading queue management
    - **Property 5: GPS Reading Queue Management**
    - **Validates: Requirements 2.5, 2.6**
    - Generate sequences of GPS readings with timestamps
    - Verify max size 500 with oldest discard, and 30-minute age filtering
    - Use property-based testing library for Android (e.g., junit-quickcheck)
  
  - [~] 11.8 Implement session management
    - Create active session screen showing route name and status
    - Add "End Session" button to stop GPS broadcasting
    - Send session end notification to backend
    - Stop foreground service on session end
    - Handle JWT expiry by stopping service and showing login screen
    - _Requirements: 2.4, 3.5_


  - [~] 11.9 Write integration tests for Driver App
    - Test login flow with API
    - Test GPS location tracking and transmission
    - Test offline queue persistence and transmission
    - Test session lifecycle
    - Test JWT expiry handling
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 3.1, 3.2, 3.5_

- [~] 12. Checkpoint - Driver App validation
  - Ensure all tests pass, verify Driver App can authenticate and broadcast GPS location. Ask the user if questions arise.

- [ ] 13. Implement Rider App (Android)
  - [~] 13.1 Set up Android project structure
    - Create Android project with Kotlin
    - Add dependencies: Retrofit, OkHttp, Scarlet (WebSocket), Google Maps SDK
    - Set up MVVM architecture with ViewModel and LiveData
    - Configure permissions: INTERNET, ACCESS_NETWORK_STATE
    - Add Google Maps API key to manifest
    - _Requirements: 1.1, 1.3_
  
  - [~] 13.2 Implement map view with Google Maps
    - Create main activity with Google Maps fragment
    - Initialize map with default location (Jagannath University)
    - Configure map settings (zoom controls, location button)
    - _Requirements: 1.1_
  
  - [~] 13.3 Implement initial bus position fetching
    - Fetch all active bus positions from backend on app open
    - Parse response and extract bus locations
    - Handle fetch errors with error message and retry button
    - Display "No buses are currently active" message if no buses
    - _Requirements: 1.1, 1.8_
  
  - [~] 13.4 Implement bus marker rendering on map
    - Create custom marker icons for buses
    - Add markers to map for each active bus
    - Label markers with route name
    - Update marker positions when new data received
    - _Requirements: 1.1, 1.7_


  - [~] 13.5 Write property test for bus marker labeling
    - **Property 8: Bus Marker Labeling (Rider portion)**
    - **Validates: Requirements 1.7**
    - Generate random bus data with optional routeName
    - Verify label includes routeName or "No Route" if absent
    - Use property-based testing library for Android
  
  - [~] 13.6 Implement WebSocket connection for real-time updates
    - Connect to `/ws/location` WebSocket endpoint
    - Parse incoming location update messages
    - Update bus marker positions on map within 3 seconds
    - Handle connection errors and disconnections
    - _Requirements: 1.2, 1.3_
  
  - [~] 13.7 Implement network connectivity monitoring
    - Detect network connectivity changes using ConnectivityManager
    - Display "Offline — showing last known location" indicator when offline
    - Show indicator persistently for entire offline duration
    - Dismiss indicator only when connectivity restored
    - Distinguish between network loss and other data unavailability
    - _Requirements: 1.4_
  
  - [~] 13.8 Implement WebSocket reconnection logic
    - Detect WebSocket disconnection while network is available
    - Display "Reconnecting…" indicator during reconnection attempts
    - Retry connection up to 3 times at 5-second intervals
    - Refresh all bus positions on successful reconnection
    - _Requirements: 1.5, 1.6_
  
  - [~] 13.9 Write integration tests for Rider App
    - Test initial bus position fetching and display
    - Test WebSocket connection and real-time updates
    - Test offline indicator display and dismissal
    - Test reconnection logic
    - Test error handling and retry
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8_

- [~] 14. Checkpoint - Rider App validation
  - Ensure all tests pass, verify Rider App displays live bus locations with offline resilience. Ask the user if questions arise.


- [ ] 15. Implement Admin Panel (React)
  - [~] 15.1 Set up React project structure
    - Create React app with TypeScript using Vite
    - Add dependencies: React Router, Axios, React Query, Leaflet (map library)
    - Set up folder structure: components, pages, services, hooks, types
    - Configure environment variables for API base URL
    - _Requirements: 4.1, 5.1, 7.1_
  
  - [~] 15.2 Implement login page and authentication
    - Create login form with username and password fields
    - Implement login API call to `/auth/login` with admin role
    - Store JWT in localStorage or sessionStorage
    - Handle authentication errors (401, 422, 429, 503)
    - Redirect to dashboard on successful login
    - _Requirements: 4.1, 4.2, 4.3_
  
  - [~] 15.3 Implement protected route wrapper
    - Create ProtectedRoute component checking for valid JWT
    - Redirect to login page if JWT missing or expired
    - Add JWT to Authorization header for all API requests
    - Handle 403 errors by redirecting to login
    - _Requirements: 4.4, 4.5_
  
  - [~] 15.4 Implement route management page
    - Create route list view displaying all routes with stops
    - Add "Create Route" button opening route form modal
    - Add "Edit" and "Delete" buttons for each route
    - Display assigned bus and driver information
    - _Requirements: 5.1, 5.2, 5.4, 5.5_
  
  - [~] 15.5 Implement route creation form
    - Create form with route name input and dynamic stop list
    - Allow adding/removing stops with name and coordinates
    - Validate route name (1-100 chars) and stop count (2-50)
    - Validate stop names (1-100 chars) and coordinates (lat -90 to 90, lon -180 to 180)
    - Display validation errors inline on form fields
    - Submit to POST `/routes` endpoint
    - Handle errors (409 for duplicate name, 422 for validation)
    - Refresh route list on success
    - _Requirements: 5.1, 5.7_


  - [~] 15.6 Implement route update form
    - Pre-populate form with existing route data
    - Allow editing route name and stop list
    - Validate input same as creation form
    - Submit to PUT `/routes/:id` endpoint
    - Handle errors (404, 409, 422)
    - Implement optimistic UI update with rollback on error
    - _Requirements: 5.2, 5.3, 5.4_
  
  - [~] 15.7 Implement route deletion with confirmation
    - Show confirmation modal before deletion
    - Submit to DELETE `/routes/:id` endpoint
    - Handle errors (404, 409 for active bus)
    - Display error message if route has active bus
    - Refresh route list on success
    - _Requirements: 5.5, 5.6_
  
  - [~] 15.8 Implement bus assignment interface
    - Create bus assignment form for each route
    - Fetch available buses and drivers
    - Allow selecting bus and driver for route
    - Submit to POST `/routes/:id/assign` endpoint
    - Handle errors (404 for missing bus/driver)
    - Update route list on success
    - _Requirements: 5.8_
  
  - [~] 15.9 Implement live map view for admins
    - Create map page with Leaflet map component
    - Fetch initial bus positions on page load
    - Display bus markers on map
    - Label markers with driver name and route name
    - Use "Unassigned" for missing driver name, "No Route" for missing route name
    - _Requirements: 7.1, 7.4_
  
  - [~] 15.10 Write property test for bus marker labeling
    - **Property 8: Bus Marker Labeling (Admin portion)**
    - **Validates: Requirements 7.4**
    - Generate random bus data with optional driverName and routeName
    - Verify label includes driverName or "Unassigned", routeName or "No Route"
    - Use property-based testing library for TypeScript (e.g., fast-check)


  - [~] 15.11 Implement WebSocket connection for admin live map
    - Connect to `/ws/location` WebSocket endpoint with admin JWT
    - Parse incoming location update messages
    - Update bus marker positions on map within 3 seconds
    - Handle connection errors and disconnections
    - _Requirements: 7.2, 7.3_
  
  - [~] 15.12 Implement offline indicator for admin map
    - Detect network connectivity loss
    - Display "Offline — showing last known location" indicator persistently
    - Dismiss indicator only when connectivity restored
    - Resume WebSocket connection and refresh positions on reconnection
    - _Requirements: 7.5_
  
  - [~] 15.13 Write integration tests for Admin Panel
    - Test login flow with API
    - Test route CRUD operations
    - Test form validation and error handling
    - Test bus assignment
    - Test live map WebSocket connection and updates
    - Test offline indicator
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 7.1, 7.2, 7.3, 7.4, 7.5_

- [~] 16. Checkpoint - Admin Panel validation
  - Ensure all tests pass, verify Admin Panel can manage routes and view live bus locations. Ask the user if questions arise.

- [ ] 17. Implement deployment configuration
  - [~] 17.1 Create Docker Compose for local development
    - Create docker-compose.yml with PostgreSQL, Redis, and all services
    - Configure service dependencies and networking
    - Add volume mounts for persistent data
    - Set up environment variables
    - _Requirements: 9.4, 9.5_
  
  - [~] 17.2 Create Railway deployment configuration
    - Create railway.json for each service
    - Configure resource limits (RAM, CPU) per service
    - Set up environment variable templates
    - Configure health check endpoints
    - _Requirements: 9.5_


  - [~] 17.3 Set up Neon PostgreSQL database
    - Create Neon project and database
    - Run database migrations
    - Seed initial driver and admin accounts
    - Configure connection pooling
    - _Requirements: 9.4_
  
  - [~] 17.4 Set up Upstash Redis instance
    - Create Upstash Redis database
    - Configure TLS connection
    - Test pub/sub functionality
    - _Requirements: 9.4_
  
  - [~] 17.5 Deploy services to Railway
    - Deploy API Gateway, Auth Service, Location Service, Route Service
    - Configure environment variables with database and Redis URLs
    - Set up custom domains (optional)
    - Verify health check endpoints
    - _Requirements: 9.5_
  
  - [~] 17.6 Create GitHub Actions CI/CD pipeline
    - Set up workflow for running tests on every commit
    - Run unit tests, property tests, and integration tests
    - Run linting (golangci-lint) and formatting checks (gofmt)
    - Run security scanning (gosec)
    - Deploy to Railway on main branch push
    - _Requirements: 9.5_

- [ ] 18. Implement load and performance testing
  - [~] 18.1 Write load tests for Location Service
    - Simulate 30 concurrent driver connections sending GPS updates every 5 seconds
    - Simulate 300 concurrent rider WebSocket connections
    - Measure broadcast latency (target < 2 seconds)
    - Verify connection rejection at limits (503 for HTTP, close code 1013 for WebSocket)
    - _Requirements: 9.1, 9.2, 9.3_
  
  - [~] 18.2 Write load tests for API Gateway
    - Simulate 1000 requests/second across all endpoints
    - Measure routing latency and JWT validation overhead
    - Verify rate limiting enforcement
    - _Requirements: 8.3, 9.1_


  - [~] 18.3 Write database performance tests
    - Test route queries with 50 stops
    - Test concurrent route updates
    - Measure query execution time and connection pool usage
    - _Requirements: 5.1, 5.2, 9.4_

- [ ] 19. Final integration and end-to-end testing
  - [~] 19.1 Write end-to-end tests for complete user flows
    - Test driver login → route selection → GPS broadcasting → rider sees location
    - Test admin login → create route → assign bus → view on live map
    - Test offline scenarios: driver loses connection → queues readings → reconnects → transmits
    - Test rider offline → shows last known position → reconnects → updates
    - _Requirements: All_
  
  - [~] 19.2 Verify system scalability within free-tier limits
    - Test with 30 concurrent drivers and 300 concurrent riders
    - Monitor resource usage (RAM, CPU, database connections, Redis commands)
    - Verify operation within free-tier limits (Neon 0.5 GB storage, Upstash 256 MB, Railway 512 MB RAM per service)
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_
  
  - [~] 19.3 Create system documentation
    - Write README with setup instructions
    - Document API endpoints with examples
    - Create deployment guide for Railway
    - Document environment variables
    - Add architecture diagrams
    - _Requirements: All_

- [~] 20. Final checkpoint - System validation
  - Ensure all tests pass, verify complete system works end-to-end with all components integrated. Ask the user if questions arise.


## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP delivery
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at key milestones
- Property tests validate universal correctness properties using gopter (Go), junit-quickcheck (Android), and fast-check (TypeScript)
- Integration tests validate infrastructure concerns (WebSocket, database, Redis pub/sub) using testcontainers-go
- The implementation follows a bottom-up approach: infrastructure → services → clients
- Backend services use Go 1.21+ with Gin framework
- Android apps use Kotlin with MVVM architecture
- Admin Panel uses React 18 with TypeScript
- All services are containerized with Docker and deployed to Railway
- PostgreSQL on Neon free tier for persistent storage
- Redis on Upstash free tier for caching and pub/sub
- System designed to operate within $0-$5/month cost constraints

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3"] },
    { "id": 1, "tasks": ["1.4", "1.5"] },
    { "id": 2, "tasks": ["2.1", "2.2", "4.1", "6.1", "9.1"] },
    { "id": 3, "tasks": ["2.3", "2.4", "4.2", "6.2", "9.2"] },
    { "id": 4, "tasks": ["2.5", "4.3", "4.4", "6.3", "6.4", "9.3", "9.4"] },
    { "id": 5, "tasks": ["2.6", "2.7", "4.5", "6.5", "6.6", "9.5", "9.6", "9.7"] },
    { "id": 6, "tasks": ["4.6", "4.7", "4.8", "6.7", "7.1", "9.8", "9.9"] },
    { "id": 7, "tasks": ["4.9", "7.2", "7.3", "9.10"] },
    { "id": 8, "tasks": ["7.4", "9.11"] },
    { "id": 9, "tasks": ["11.1", "13.1", "15.1"] },
    { "id": 10, "tasks": ["11.2", "11.3", "13.2", "15.2"] },
    { "id": 11, "tasks": ["11.4", "11.5", "13.3", "13.4", "15.3", "15.4"] },
    { "id": 12, "tasks": ["11.6", "11.8", "13.5", "13.6", "15.5", "15.6"] },
    { "id": 13, "tasks": ["11.7", "11.9", "13.7", "13.8", "15.7", "15.8"] },
    { "id": 14, "tasks": ["13.9", "15.9", "15.10"] },
    { "id": 15, "tasks": ["15.11", "15.12"] },
    { "id": 16, "tasks": ["15.13"] },
    { "id": 17, "tasks": ["17.1", "17.2"] },
    { "id": 18, "tasks": ["17.3", "17.4"] },
    { "id": 19, "tasks": ["17.5", "17.6"] },
    { "id": 20, "tasks": ["18.1", "18.2", "18.3"] },
    { "id": 21, "tasks": ["19.1", "19.2"] },
    { "id": 22, "tasks": ["19.3"] }
  ]
}
```
