# ⚡ Quick Reference - JnU Live Bus Tracker

---

## 🚀 Quick Start Commands

### Start Backend Services
```bash
# Terminal 1
redis-server

# Terminal 2
cd auth-service && go run .

# Terminal 3
cd route-service && go run .

# Terminal 4
cd location-service && go run .

# Terminal 5
cd api-gateway && go run .
```

### Start Frontend Apps
```bash
# Driver App
cd driver_app && flutter run

# Rider App
cd rider_app && flutter run

# Admin Panel
cd admin-panel && npm run dev
```

---

## 📍 Service URLs

| Service | URL | Port |
|---------|-----|------|
| API Gateway | http://localhost:8080 | 8080 |
| Auth Service | http://localhost:8081 | 8081 |
| Route Service | http://localhost:8083 | 8083 |
| Location Service | http://localhost:8082 | 8082 |
| WebSocket | ws://localhost:8080/ws/location | 8080 |
| Admin Panel | http://localhost:5173 | 5173 |

---

## 🔑 Test Credentials

```bash
# Admin
Username: admin
Password: admin123
Role: admin

# Driver
Username: driver1
Password: driver123
Role: driver
```

---

## 📡 API Endpoints

### Authentication
```bash
POST /auth/login
Body: {"username": "admin", "password": "admin123", "role": "admin"}
```

### Routes
```bash
GET    /routes                      # List all routes
POST   /routes                      # Create route
PUT    /routes/:id                  # Update route
DELETE /routes/:id                  # Delete route
POST   /routes/:id/assign           # Assign bus to route
```

### Location
```bash
POST /location/update               # Send GPS update
GET  /location/buses                # Get all active buses
GET  /ws/location                   # WebSocket connection
```

---

## 🧪 cURL Test Commands

### Login (Admin)
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123","role":"admin"}'
```

### Create Route
```bash
curl -X POST http://localhost:8080/routes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "Test Route",
    "stops": [
      {"name": "Stop 1", "latitude": 23.71, "longitude": 90.40},
      {"name": "Stop 2", "latitude": 23.72, "longitude": 90.41}
    ]
  }'
```

### Send GPS Update
```bash
curl -X POST http://localhost:8080/location/update \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_DRIVER_JWT_TOKEN" \
  -d '{
    "busId": "bus-001",
    "latitude": 23.7104,
    "longitude": 90.4074,
    "timestamp": "2026-06-02T10:30:00Z",
    "routeId": "route-001",
    "routeName": "Main Route"
  }'
```

### WebSocket Test
```bash
npm install -g wscat
wscat -c ws://localhost:8080/ws/location
```

---

## 🗂️ Key Files

### Backend
- `api-gateway/main.go` - API Gateway
- `auth-service/service.go` - Login logic
- `route-service/main.go` - Route CRUD
- `location-service/handler.go` - GPS processing
- `shared/jwt/jwt.go` - JWT utilities
- `shared/redis/client.go` - Redis client
- `migrations/*.sql` - Database schema

### Mobile Apps
- `driver_app/lib/config/app_config.dart` - Configuration
- `rider_app/lib/config/app_config.dart` - Configuration

### Admin Panel
- `admin-panel/src/config/api.ts` - API configuration
- `admin-panel/src/services/apiService.ts` - API service

---

## 🔧 Configuration Files

### Auth Service (.env)
```env
PORT=8081
DATABASE_URL=postgresql://user:pass@localhost:5432/bus_tracker
JWT_SECRET=your-secret-key
REDIS_URL=redis://localhost:6379
DRIVER_TOKEN_EXPIRY=12h
ADMIN_TOKEN_EXPIRY=8h
```

### Location Service (.env)
```env
PORT=8082
REDIS_URL=redis://localhost:6379
MAX_DRIVER_CONNECTIONS=30
MAX_RIDER_CONNECTIONS=300
LOCATION_TTL=30m
MIN_UPDATE_INTERVAL=5s
```

### API Gateway (.env)
```env
PORT=8080
AUTH_SERVICE_URL=http://localhost:8081
LOCATION_SERVICE_URL=http://localhost:8082
ROUTE_SERVICE_URL=http://localhost:8083
JWT_SECRET=your-secret-key
```

---

## 📊 System Limits

| Metric | Value |
|--------|-------|
| Max Concurrent Drivers | 30 |
| Max Concurrent Riders | 300 |
| GPS Update Interval | 5 seconds |
| Location TTL | 30 minutes |
| JWT Expiry (Driver) | 12 hours |
| JWT Expiry (Admin) | 8 hours |
| Rate Limit (Admin) | 5/minute |
| Rate Limit (Driver) | 10/minute |
| Max Stops per Route | 50 |
| Min Stops per Route | 2 |
| Offline Queue Size | 500 readings |

---

## 🐛 Troubleshooting

### "Connection refused"
```bash
# Check if service is running
curl http://localhost:8080/health

# Check logs
go run . 2>&1 | tee service.log
```

### "JWT invalid"
```bash
# Check JWT_SECRET matches in:
# - auth-service/.env
# - api-gateway/.env
```

### "Database connection failed"
```bash
# Check PostgreSQL is running
pg_isready

# Test connection
psql -h localhost -U postgres -d bus_tracker
```

### "Redis connection failed"
```bash
# Check Redis is running
redis-cli ping

# Should return: PONG
```

### "WebSocket not connecting"
```bash
# Check Location Service is running
curl http://localhost:8082/health

# Check API Gateway is proxying
curl http://localhost:8080/health
```

---

## 📚 Documentation Links

| Guide | Purpose |
|-------|---------|
| [GETTING_STARTED.md](GETTING_STARTED.md) | Quick start |
| [SKELETON_COMPLETE.md](SKELETON_COMPLETE.md) | Complete overview |
| [MOBILE_APPS_IMPLEMENTATION_GUIDE.md](MOBILE_APPS_IMPLEMENTATION_GUIDE.md) | Flutter implementation |
| [ADMIN_PANEL_IMPLEMENTATION_GUIDE.md](ADMIN_PANEL_IMPLEMENTATION_GUIDE.md) | React implementation |
| [FINAL_IMPLEMENTATION_SUMMARY.md](FINAL_IMPLEMENTATION_SUMMARY.md) | Final summary |
| [VISUAL_SUMMARY.md](VISUAL_SUMMARY.md) | Visual diagrams |
| [README.md](README.md) | Project overview |

---

## ✅ Health Check Checklist

```bash
# All should return {"status":"healthy"}
curl http://localhost:8080/health  # API Gateway
curl http://localhost:8081/health  # Auth Service
curl http://localhost:8082/health  # Location Service
curl http://localhost:8083/health  # Route Service

# Redis
redis-cli ping                     # Should return PONG

# PostgreSQL
pg_isready                         # Should return "accepting connections"
```

---

## 🚀 Deployment Checklist

- [ ] Backend services running locally
- [ ] All health checks passing
- [ ] Database migrations applied
- [ ] Redis connected
- [ ] JWT tokens working
- [ ] WebSocket broadcasting working
- [ ] Mobile apps connect to backend
- [ ] Admin panel connects to backend
- [ ] Create Neon PostgreSQL database
- [ ] Create Upstash Redis instance
- [ ] Deploy to Railway
- [ ] Update API URLs in apps
- [ ] Test production endpoints
- [ ] Build mobile apps
- [ ] Deploy admin panel
- [ ] Launch! 🎉

---

**Last Updated**: June 2, 2026
**Status**: Reference Guide
