# University Bus Tracker - Complete Build Guide

## 🎯 Project Overview

A real-time bus tracking system for Jagannath University with:
- **4 Go Microservices**: API Gateway, Auth Service, Location Service, Route Service
- **2 Android Apps**: Driver App (GPS broadcasting), Rider App (live map)
- **1 React Admin Panel**: Route management and fleet monitoring
- **Cost**: $0-5/month using free tiers (Railway, Neon PostgreSQL, Upstash Redis)

## 📊 Current Status

✅ **Completed (5/106 tasks)**:
- Project structure initialized
- PostgreSQL schema and migrations created
- Redis pub/sub infrastructure set up
- JWT utilities implemented and tested
- Shared configuration packages ready

🔄 **In Progress (4 tasks)**:
- Auth Service HTTP server
- Password hashing
- Login endpoint
- Rate limiting

⏳ **Remaining (97 tasks)**:
- Complete Auth Service
- Build Route Service
- Build Location Service
- Build API Gateway
- Build Driver App (Android)
- Build Rider App (Android)
- Build Admin Panel (React)
- Deployment configuration
- Testing and documentation

## 🚀 Quick Start

### Prerequisites

```bash
# Install Go 1.21+
go version

# Install Docker
docker --version

# Install PostgreSQL client (for migrations)
psql --version

# Install Redis client (for testing)
redis-cli --version

# For Android apps
# - Android Studio with SDK 33+
# - Google Maps API key

# For React admin panel
node --version  # v18+
npm --version   # v9+
```

### 1. Environment Setup

```bash
cd /Users/mdasifurrahmansikder/mern-ecommerce

# Copy environment templates
cp api-gateway/.env.example api-gateway/.env
cp auth-service/.env.example auth-service/.env
cp location-service/.env.example location-service/.env
cp route-service/.env.example route-service/.env

# Generate JWT secret
openssl rand -base64 32

# Edit .env files with your values:
# - JWT_SECRET (from above)
# - DATABASE_URL (Neon PostgreSQL)
# - REDIS_URL (Upstash Redis)
```

### 2. Database Setup

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
export DATABASE_URL="postgresql://user:pass@host:5432/bustrack"
migrate -path ./migrations -database $DATABASE_URL up

# Seed initial data (create admin and driver accounts)
psql $DATABASE_URL < migrations/seed.sql
```

### 3. Start Services Locally

```bash
# Start PostgreSQL and Redis with Docker
docker-compose up -d postgres redis

# Start Auth Service
cd auth-service
go run main.go

# Start Route Service (new terminal)
cd route-service
go run main.go

# Start Location Service (new terminal)
cd location-service
go run main.go

# Start API Gateway (new terminal)
cd api-gateway
go run main.go
```

### 4. Test the Backend

```bash
# Test Auth Service
curl -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123","role":"admin"}'

# Test Route Service
curl http://localhost:8083/routes

# Test Location Service health
curl http://localhost:8082/health
```

## 📁 Project Structure

```
mern-ecommerce/
├── api-gateway/           # API Gateway (Port 8080)
│   ├── main.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── ratelimit.go
│   │   └── routing.go
│   ├── Dockerfile
│   └── .env.example
│
├── auth-service/          # Auth Service (Port 8081)
│   ├── main.go
│   ├── handlers/
│   │   └── auth.go
│   ├── models/
│   │   └── user.go
│   ├── services/
│   │   ├── password.go
│   │   └── ratelimit.go
│   ├── Dockerfile
│   └── .env.example
│
├── location-service/      # Location Service (Port 8082)
│   ├── main.go
│   ├── handlers/
│   │   ├── location.go
│   │   └── websocket.go
│   ├── services/
│   │   ├── broadcast.go
│   │   ├── validation.go
│   │   └── ratelimit.go
│   ├── Dockerfile
│   └── .env.example
│
├── route-service/         # Route Service (Port 8083)
│   ├── main.go
│   ├── handlers/
│   │   └── routes.go
│   ├── models/
│   │   ├── route.go
│   │   └── stop.go
│   ├── services/
│   │   ├── validation.go
│   │   └── pubsub.go
│   ├── Dockerfile
│   └── .env.example
│
├── shared/                # Shared utilities
│   ├── config/
│   ├── jwt/              # ✅ JWT utilities (complete)
│   ├── redis/            # ✅ Redis client (complete)
│   ├── types/
│   └── utils/
│
├── migrations/            # ✅ Database migrations (complete)
│   ├── 000001_create_users_table.up.sql
│   ├── 000002_create_routes_table.up.sql
│   ├── 000003_create_stops_table.up.sql
│   ├── 000004_create_buses_table.up.sql
│   ├── 000005_create_route_assignments_table.up.sql
│   └── seed.sql
│
├── driver-app/            # Android Driver App
│   ├── app/
│   │   └── src/main/
│   │       ├── java/
│   │       └── res/
│   └── build.gradle
│
├── rider-app/             # Android Rider App
│   ├── app/
│   │   └── src/main/
│   │       ├── java/
│   │       └── res/
│   └── build.gradle
│
├── admin-panel/           # React Admin Panel
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── services/
│   │   └── App.tsx
│   ├── package.json
│   └── vite.config.ts
│
├── docker-compose.yml     # Local development
├── .kiro/specs/          # ✅ Requirements & Design docs
└── BUILD_GUIDE.md        # This file
```

## 🔨 Implementation Roadmap

### Phase 1: Backend Services (Tasks 2-10) - ~2-3 hours

#### Auth Service (Tasks 2.1-2.7)
```bash
# What needs to be built:
# - HTTP server with Gin framework
# - Login endpoint (/auth/login)
# - Password hashing with bcrypt
# - JWT generation (already have utilities)
# - Rate limiting for failed attempts
# - Database integration with GORM

# Key files to create:
auth-service/main.go
auth-service/handlers/auth.go
auth-service/models/user.go
auth-service/services/password.go
auth-service/services/ratelimit.go
```

#### Route Service (Tasks 4.1-4.9)
```bash
# What needs to be built:
# - HTTP server with Gin
# - CRUD endpoints for routes
# - Stop management
# - Bus assignment
# - Redis pub/sub for route updates
# - Input validation

# Key files to create:
route-service/main.go
route-service/handlers/routes.go
route-service/models/route.go
route-service/models/stop.go
route-service/services/validation.go
route-service/services/pubsub.go
```

#### Location Service (Tasks 6.1-7.4)
```bash
# What needs to be built:
# - HTTP server with Gin
# - GPS coordinate validation
# - Rate limiting per driver
# - WebSocket server for real-time updates
# - Redis caching for last-known positions
# - Broadcast to 300+ concurrent connections

# Key files to create:
location-service/main.go
location-service/handlers/location.go
location-service/handlers/websocket.go
location-service/services/validation.go
location-service/services/broadcast.go
location-service/services/ratelimit.go
```

#### API Gateway (Tasks 9.1-9.11)
```bash
# What needs to be built:
# - HTTP server with Gin
# - Request routing middleware
# - JWT validation middleware
# - CORS middleware
# - Role-based access control
# - WebSocket proxy to Location Service
# - Rate limiting for failed JWT attempts

# Key files to create:
api-gateway/main.go
api-gateway/middleware/auth.go
api-gateway/middleware/cors.go
api-gateway/middleware/ratelimit.go
api-gateway/middleware/routing.go
api-gateway/middleware/rbac.go
```

### Phase 2: Mobile Apps (Tasks 11-14) - ~2-3 hours

#### Driver App (Tasks 11.1-11.9)
```bash
# What needs to be built:
# - Android project with Kotlin
# - Login screen
# - Route selection
# - GPS tracking service (foreground)
# - Offline queue (Room database)
# - Session management

# Key files to create:
driver-app/app/src/main/java/com/jnu/bustrack/driver/
  ├── ui/LoginActivity.kt
  ├── ui/RouteSelectionActivity.kt
  ├── ui/ActiveSessionActivity.kt
  ├── service/LocationTrackingService.kt
  ├── repository/LocationRepository.kt
  ├── database/LocationDatabase.kt
  └── viewmodel/DriverViewModel.kt
```

#### Rider App (Tasks 13.1-13.9)
```bash
# What needs to be built:
# - Android project with Kotlin
# - Google Maps integration
# - WebSocket connection
# - Real-time bus markers
# - Offline indicator
# - Reconnection logic

# Key files to create:
rider-app/app/src/main/java/com/jnu/bustrack/rider/
  ├── ui/MapActivity.kt
  ├── service/WebSocketService.kt
  ├── repository/BusRepository.kt
  └── viewmodel/MapViewModel.kt
```

### Phase 3: Admin Panel (Tasks 15-16) - ~1-2 hours

#### React Admin Panel (Tasks 15.1-15.13)
```bash
# What needs to be built:
# - React + TypeScript + Vite project
# - Login page
# - Route management (CRUD)
# - Bus assignment interface
# - Live map with Leaflet
# - WebSocket connection

# Key files to create:
admin-panel/src/
  ├── pages/LoginPage.tsx
  ├── pages/RoutesPage.tsx
  ├── pages/LiveMapPage.tsx
  ├── components/RouteForm.tsx
  ├── components/BusMarker.tsx
  ├── services/api.ts
  ├── services/websocket.ts
  └── hooks/useWebSocket.ts
```

### Phase 4: Deployment (Tasks 17-20) - ~1 hour

```bash
# What needs to be done:
# 1. Create Railway project
# 2. Deploy 4 Go services
# 3. Set up Neon PostgreSQL
# 4. Set up Upstash Redis
# 5. Configure environment variables
# 6. Set up CI/CD with GitHub Actions
# 7. Create documentation
```

## 🎯 Next Steps

### Option 1: Continue with Kiro (Recommended)
Open the tasks.md file in your editor and use Kiro's task execution:
1. Click "Start task" next to task 2.1
2. Kiro will implement each task sequentially
3. Review and test after each checkpoint

### Option 2: Manual Implementation
Use this guide to implement services yourself:
1. Start with Auth Service (simplest)
2. Then Route Service
3. Then Location Service (most complex)
4. Then API Gateway
5. Finally mobile/web apps

### Option 3: Hybrid Approach
- Use Kiro for backend services (Go code)
- Implement mobile/web apps manually (if you prefer)

## 📚 Key Resources

### Documentation
- Requirements: `.kiro/specs/university-bus-tracker/requirements.md`
- Design: `.kiro/specs/university-bus-tracker/design.md`
- Tasks: `.kiro/specs/university-bus-tracker/tasks.md`

### External Services
- **Railway**: https://railway.app (deployment)
- **Neon**: https://neon.tech (PostgreSQL)
- **Upstash**: https://upstash.com (Redis)
- **Google Maps API**: https://console.cloud.google.com

### Libraries & Frameworks
- **Gin**: https://gin-gonic.com (Go web framework)
- **GORM**: https://gorm.io (Go ORM)
- **gorilla/websocket**: https://github.com/gorilla/websocket
- **React**: https://react.dev
- **Leaflet**: https://leafletjs.com (maps)

## 🐛 Troubleshooting

### Database Connection Issues
```bash
# Test connection
psql $DATABASE_URL -c "SELECT 1"

# Check migrations
migrate -path ./migrations -database $DATABASE_URL version
```

### Redis Connection Issues
```bash
# Test connection
redis-cli -u $REDIS_URL ping
```

### Port Conflicts
```bash
# Check if ports are in use
lsof -i :8080  # API Gateway
lsof -i :8081  # Auth Service
lsof -i :8082  # Location Service
lsof -i :8083  # Route Service
```

## 📞 Support

If you encounter issues:
1. Check the design document for API specifications
2. Review the requirements document for acceptance criteria
3. Check task descriptions in tasks.md
4. Ask Kiro to implement specific tasks

## 🎉 Success Criteria

Your system is complete when:
- ✅ All 4 backend services are running
- ✅ Driver can login and broadcast GPS
- ✅ Rider can see live bus locations on map
- ✅ Admin can manage routes via web panel
- ✅ System handles 30 drivers + 300 riders concurrently
- ✅ Deployed to Railway within $0-5/month budget

---

**Total Estimated Time**: 6-10 hours for complete implementation
**Current Progress**: 5% complete (infrastructure ready)
**Next Milestone**: Complete Auth Service (Tasks 2.1-2.7)
