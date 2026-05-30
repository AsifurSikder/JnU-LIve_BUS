# Requirements Document

## Introduction

The University Bus Tracker is a live bus tracking system for Jagannath University's transportation department. It allows students to see where university buses are in real time on a map. Bus drivers use a mobile app that continuously sends GPS coordinates to the backend. Admins manage routes and stops through a web panel. The system is built on a microservices architecture (Go + Gin), deployed on Railway at near-zero cost, and uses PostgreSQL (Neon) and Redis (Upstash) for persistence and caching.

The MVP covers: live bus location tracking, fixed route/stop management, driver GPS broadcasting, student map view with last-known-position fallback, and an admin panel for route management. AI-based ETA and push notifications are deferred to a later phase.

---

## Glossary

- **Rider**: A university student who views live bus locations via the mobile app.
- **Driver**: A university bus driver whose phone sends continuous GPS data. Driver accounts are pre-created by the developer.
- **Admin**: A member of the Jagannath University transportation department who manages routes and stops via the web panel.
- **Bus**: A physical university bus associated with a driver and a route.
- **Route**: A fixed, named path defined by an ordered sequence of stops, managed by the transportation department.
- **Stop**: A named geographic waypoint on a route with a fixed latitude/longitude.
- **Location_Service**: The backend microservice responsible for receiving, storing, and broadcasting real-time GPS coordinates.
- **Route_Service**: The backend microservice responsible for managing routes and stops.
- **Auth_Service**: The backend microservice responsible for authenticating drivers and admins using JWT.
- **API_Gateway**: The Go/Gin service that routes HTTP and WebSocket requests to the appropriate microservice.
- **Driver_App**: The mobile application used by drivers to broadcast GPS location.
- **Rider_App**: The Android mobile application used by students to view live bus locations.
- **Admin_Panel**: The React web application used by admins to manage routes, stops, and buses.
- **JWT**: JSON Web Token used for stateless authentication.
- **Last Known Position**: The most recently received GPS coordinate for a bus, cached in Redis.

---

## Requirements

### Requirement 1: Student Live Map View

**User Story:** As a student, I want to see all active buses on a live map, so that I can know where my bus is right now.

#### Acceptance Criteria

1. WHEN a Rider opens the Rider_App, THE Rider_App SHALL display a map showing the last known position of every active bus; IF no active buses exist, THE Rider_App SHALL display the map with no markers and an informational message stating "No buses are currently active."
2. WHEN a bus's GPS position is updated, THE Rider_App SHALL update that bus's marker on the map within 3 seconds of the update being received by the server.
3. WHILE a Rider is viewing the map, THE Rider_App SHALL maintain a WebSocket connection to the API_Gateway to receive real-time location updates.
4. IF the Rider_App loses its network connection, THEN THE Rider_App SHALL display the last known position of each bus along with an "Offline — showing last known location" indicator that is persistently visible on screen for the entire duration of the offline state, dismissed only when connectivity is restored. THE Rider_App SHALL only show this indicator when network connectivity is actually lost, not when bus data is temporarily unavailable for other reasons.
5. IF the Rider_App's WebSocket connection to the API_Gateway is dropped while network connectivity is still available, THEN THE Rider_App SHALL automatically attempt to reconnect up to 3 times at intervals of no more than 5 seconds, displaying a "Reconnecting…" indicator during each attempt.
6. WHEN the Rider_App reconnects to the network, THE Rider_App SHALL resume the WebSocket connection and refresh all bus positions.
7. THE Rider_App SHALL display each bus marker labeled with the bus's associated route name.
8. IF the Rider_App fails to fetch initial bus positions when the app opens, THEN THE Rider_App SHALL display an error message and provide a visible retry action the Rider can tap to re-attempt the fetch.

---

### Requirement 2: Driver GPS Broadcasting

**User Story:** As a driver, I want my phone to automatically send my GPS location while I am on duty, so that students can track my bus in real time.

#### Acceptance Criteria

1. WHEN a Driver logs in and selects a route, THE Driver_App SHALL begin sending the device's GPS coordinates to the Location_Service at an interval of no less than 1 second and no more than 5 seconds.
2. THE Location_Service SHALL enforce the send interval as a hard limit: IF a GPS update from a driver session arrives less than 5 seconds after the previous accepted update from that session, THEN THE Location_Service SHALL reject the update and return HTTP status 429.
3. WHILE a driver session is active, THE Driver_App SHALL continuously send GPS coordinates whether the app is in the foreground or background, without requiring manual action from the driver.
4. WHEN a Driver ends their session, THE Driver_App SHALL stop sending GPS coordinates and notify the Location_Service that the bus is no longer active.
5. IF the Driver_App loses network connectivity while a session is active, THEN THE Driver_App SHALL queue GPS readings locally in order of receipt, up to a maximum of 500 readings, and transmit them in order once connectivity is restored; queued readings older than 30 minutes SHALL be discarded before transmission.
6. IF the local queue reaches 500 readings, THE Driver_App SHALL discard the oldest reading before enqueuing each new reading.
7. THE Driver_App SHALL require a valid JWT to send location data to the Location_Service.

---

### Requirement 3: Driver Authentication

**User Story:** As a driver, I want to log in with credentials pre-set by the developer, so that only authorized drivers can broadcast location data.

#### Acceptance Criteria

1. WHEN a Driver submits valid credentials, THE Auth_Service SHALL return a signed JWT with a 12-hour expiry and HTTP status 200.
2. IF a Driver submits invalid credentials, THEN THE Auth_Service SHALL return an error response with HTTP status 401.
3. IF the Auth_Service is unavailable when a Driver submits credentials, THEN THE Auth_Service SHALL return HTTP status 503.
4. THE Auth_Service SHALL store driver credentials as bcrypt-hashed passwords in PostgreSQL.
5. WHEN a Driver's JWT expires, THE Driver_App SHALL immediately stop GPS broadcasting and display the login screen, prompting the driver to log in again before resuming GPS broadcasting.

---

### Requirement 4: Admin Authentication

**User Story:** As an admin, I want to log in to the admin panel, so that I can manage routes and stops securely.

#### Acceptance Criteria

1. WHEN an Admin submits valid credentials, THE Auth_Service SHALL return a signed JWT scoped to the admin role with an 8-hour expiry and HTTP status 200. THE Auth_Service SHALL store admin credentials as bcrypt-hashed passwords in PostgreSQL.
2. IF an Admin submits invalid credentials, THEN THE Auth_Service SHALL return an error response with HTTP status 401.
3. IF an Admin submits invalid credentials 5 or more times within a 60-second window, THEN THE Auth_Service SHALL return HTTP status 429 for all subsequent login attempts from that source IP within the same window.
4. IF a request arrives at an admin-scoped endpoint without a valid admin-role JWT, THEN THE API_Gateway SHALL return HTTP status 403 without forwarding the request to any downstream service.
5. WHEN an Admin's JWT expires, THE Admin_Panel SHALL display the login screen and prompt the admin to log in again before allowing further route management actions.

---

### Requirement 5: Route and Stop Management

**User Story:** As an admin, I want to create, update, and delete routes and stops, so that the system reflects the current transportation schedule.

#### Acceptance Criteria

1. WHEN an Admin creates a route with a unique name (1–100 characters) and an ordered list of 2–50 stops where each stop has a name (1–100 characters) and a valid latitude/longitude coordinate, THE Route_Service SHALL persist the route and its stops to PostgreSQL and return the created route with its assigned ID and HTTP status 201.
2. WHEN an Admin updates a route's name or stop list with valid data, THE Route_Service SHALL update the persisted record and return the updated route with HTTP status 200.
3. IF a route update fails input validation, THEN THE Route_Service SHALL return HTTP status 422 without modifying the existing route data.
4. IF a route update encounters a database error, THEN THE Route_Service SHALL return HTTP status 500 without modifying the existing route data.
5. WHEN an Admin deletes a route that has no active bus currently assigned, THE Route_Service SHALL remove the route and all associated stops from PostgreSQL and return HTTP status 204.
6. IF an Admin attempts to delete a route that has an active bus currently assigned, THEN THE Route_Service SHALL reject the request and return HTTP status 409.
7. IF an Admin attempts to create a route with a name that already exists, THEN THE Route_Service SHALL reject the request and return HTTP status 409 without persisting any data.
8. WHEN an Admin assigns a bus to a route, THE Route_Service SHALL associate the specified driver account with that route, replacing any existing bus assignment for that route; IF the specified bus or driver does not exist, THEN THE Route_Service SHALL return HTTP status 404.
9. WHEN a route is updated or deleted, THE Route_Service SHALL publish the change so that the Rider_App reflects the updated route information within 10 seconds.

---

### Requirement 6: Real-Time Location Processing

**User Story:** As a system operator, I want GPS updates from drivers to be processed and broadcast to riders with minimal delay, so that the map stays accurate.

#### Acceptance Criteria

1. WHEN the Location_Service receives a valid GPS coordinate from a Driver, THE Location_Service SHALL store the coordinate as the Last Known Position for that bus in Redis.
2. WHEN the Location_Service receives a valid GPS coordinate, THE Location_Service SHALL broadcast the update to all connected Rider_App WebSocket clients subscribed to that bus or route within 2 seconds.
3. IF a GPS payload is missing any of the required fields (bus ID, latitude, longitude, or UTC timestamp), THEN THE Location_Service SHALL reject the payload and return HTTP status 422.
4. IF a GPS coordinate contains a latitude outside the range [-90, 90] or a longitude outside the range [-180, 180], or a UTC timestamp that is malformed or absent, THEN THE Location_Service SHALL reject the payload and return HTTP status 422.
5. IF a GPS coordinate contains a UTC timestamp older than 5 minutes relative to the server's current time, THEN THE Location_Service SHALL reject the payload and return HTTP status 422 to prevent stale coordinates from being broadcast.
6. WHEN the Location_Service stores a GPS coordinate for a bus, THE Location_Service SHALL mark that bus as active and persist the coordinate to Redis with a TTL of 30 minutes, after which the bus SHALL be considered inactive.

---

### Requirement 7: Admin Live Map View

**User Story:** As an admin, I want to see all buses on a live map in the admin panel, so that I can monitor the fleet in real time.

#### Acceptance Criteria

1. WHEN an Admin opens the Admin_Panel map view, THE Admin_Panel SHALL display the last known position of all active buses on a map.
2. WHILE an Admin is viewing the map, THE Admin_Panel SHALL maintain a WebSocket connection to the API_Gateway to receive real-time location updates.
3. WHEN a bus position is updated, THE Admin_Panel SHALL update that bus's marker within 3 seconds of the update being received by the server.
4. THE Admin_Panel SHALL label each bus marker with the driver's name and the assigned route name; IF a driver name is unavailable, THE Admin_Panel SHALL display "Unassigned"; IF a route name is unavailable, THE Admin_Panel SHALL display "No Route".
5. IF the Admin_Panel loses its network connection, THEN THE Admin_Panel SHALL display the last known position of each bus along with a persistently visible "Offline — showing last known location" indicator, dismissed only when connectivity is restored; WHEN connectivity is restored, THE Admin_Panel SHALL resume the WebSocket connection and refresh all bus positions.

---

### Requirement 8: API Gateway Routing

**User Story:** As a developer, I want a single API gateway to route all client requests to the correct microservice, so that clients have one consistent entry point.

#### Acceptance Criteria

1. WHEN a request arrives at the API_Gateway with a URL path beginning with `/routes`, THE API_Gateway SHALL forward the request to the Route_Service; WHEN a request arrives with a URL path beginning with `/auth`, THE API_Gateway SHALL forward the request to the Auth_Service.
2. WHEN the API_Gateway receives an HTTP request with a `Connection: Upgrade` and `Upgrade: websocket` header, THE API_Gateway SHALL proxy the request to the Location_Service.
3. IF a request arrives at the API_Gateway with a malformed or expired JWT, THEN THE API_Gateway SHALL return HTTP status 401 without forwarding the request to any downstream service, log the attempt including the source IP and timestamp, and reject subsequent requests from that source IP with HTTP status 429 without forwarding if 10 or more failed JWT attempts from that IP occur within a 60-second sliding window.
4. THE API_Gateway SHALL enforce CORS by accepting requests only from the registered origins of the Admin_Panel and the Rider_App, permitting the HTTP methods GET, POST, PUT, DELETE, and OPTIONS, and rejecting requests from any other origin with HTTP status 403.
5. IF a request arrives at the API_Gateway with a URL path that does not match any configured route prefix, THEN THE API_Gateway SHALL return HTTP status 404 without forwarding the request to any downstream service.

---

### Requirement 9: System Scalability and Cost Constraints

**User Story:** As a developer, I want the system to handle up to 30 concurrent buses and hundreds of concurrent rider connections within free-tier infrastructure limits, so that operating costs remain between $0 and $5 per month.

#### Acceptance Criteria

1. THE Location_Service SHALL support at least 30 concurrent driver connections (WebSocket or HTTP) sending GPS updates simultaneously within any 60-second window without degrading response times beyond the 2-second broadcast SLA defined in Requirement 6.
2. THE Location_Service SHALL support at least 300 concurrent Rider_App WebSocket connections receiving location updates within any 60-second window.
3. IF the number of concurrent driver connections reaches 30 or the number of concurrent rider WebSocket connections reaches 300, THE Location_Service SHALL reject new connection attempts with an appropriate error response (HTTP status 503 for HTTP, WebSocket close code 1013 for WebSocket) rather than silently degrading.
4. THE system SHALL use PostgreSQL on Neon free tier (max 0.5 GB storage, 1 GB RAM) for persistent storage and Redis on Upstash free tier (max 256 MB) for caching and pub/sub.
5. THE system SHALL be deployable as Go containers on Railway's free tier, where "deployable" means each container starts successfully, passes its health check endpoint, and serves connections within the resource limits defined in criterion 4.
