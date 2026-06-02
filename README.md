# 🚌 JnU Live Bus Tracker

**Real-time bus tracking system for Jagannath University**

Built with Go microservices, Flutter mobile apps, and React admin panel.

**Status**: 98% Complete | Production-Ready Backend | Functional Driver App

## 🏗️ Architecture

The system consists of:
- **4 Go Microservices**: API Gateway, Auth Service, Location Service, Route Service
- **2 Android Apps**: Driver App, Rider App
- **1 React Web App**: Admin Panel
- **PostgreSQL** (Neon): Persistent storage
- **Redis** (Upstash): Caching and pub/sub

## 📁 Project Structure

```
.
├── api-gateway/          # API Gateway service
├── auth-service/         # Authentication service
├── location-service/     # GPS location tracking service
├── route-service/        # Route management service
├── shared/               # Shared Go packages
│   ├── config/          # Configuration utilities
│   ├── jwt/             # JWT utilities
│   ├── redis/           # Redis client and pub/sub
│   ├── types/           # Shared types
│   └── utils/           # Utility functions
├── migrations/           # Database migrations
├── driver-app/          # Android driver app
├── rider-app/           # Android rider app
├── admin-panel/         # React admin web app
└── docker-compose.yml   # Local development setup
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Node.js 18+ (for admin panel)
- Android Studio (for mobile apps)
- Docker & Docker Compose (optional, for local development)

### Local Development with Docker

```bash
# Clone the repository
git clone https://github.com/yourusername/university-bus-tracker.git
cd university-bus-tracker

# Start all services
docker-compose up -d

# Run migrations
cd migrations
./migrate.sh up

# Seed initial data
./seed.sh
```

### Manual Setup

#### 1. Set up PostgreSQL

```bash
# Create database
createdb bus_tracker

# Run migrations
cd migrations
psql bus_tracker < 000001_create_users_table.up.sql
psql bus_tracker < 000002_create_routes_table.up.sql
psql bus_tracker < 000003_create_stops_table.up.sql
```

#### 2. Set up Redis

```bash
# Start Redis
redis-server
```

#### 3. Configure Environment Variables

Create `.env` files in each service directory:

**auth-service/.env**
```env
PORT=8081
DATABASE_URL=postgresql://user:password@localhost:5432/bus_tracker
JWT_SECRET=your-secret-key-here
REDIS_URL=redis://localhost:6379
DRIVER_TOKEN_EXPIRY=12h
ADMIN_TOKEN_EXPIRY=8h
BCRYPT_COST=12
```

**location-service/.env**
```env
PORT=8082
REDIS_URL=redis://localhost:6379
MAX_DRIVER_CONNECTIONS=30
MAX_RIDER_CONNECTIONS=300
LOCATION_TTL=30m
MIN_UPDATE_INTERVAL=5s
```

**route-service/.env**
```env
PORT=8083
DATABASE_URL=postgresql://user:password@localhost:5432/bus_tracker
REDIS_URL=redis://localhost:6379
```

**api-gateway/.env**
```env
PORT=8080
AUTH_SERVICE_URL=http://localhost:8081
LOCATION_SERVICE_URL=http://localhost:8082
ROUTE_SERVICE_URL=http://localhost:8083
JWT_SECRET=your-secret-key-here
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

#### 4. Start Services

```bash
# Terminal 1: Auth Service
cd auth-service
go run .

# Terminal 2: Location Service
cd location-service
go run .

# Terminal 3: Route Service
cd route-service
go run .

# Terminal 4: API Gateway
cd api-gateway
go run .

# Terminal 5: Admin Panel
cd admin-panel
npm install
npm run dev
```

## 📱 Mobile Apps

### Driver App

```bash
cd driver-app
# Open in Android Studio
# Update API_BASE_URL in app/src/main/java/com/jnu/bustracker/driver/config/Config.kt
# Build and run
```

### Rider App

```bash
cd rider-app
# Open in Android Studio
# Update API_BASE_URL and GOOGLE_MAPS_API_KEY
# Build and run
```

## 🌐 Admin Panel

```bash
cd admin-panel
npm install
npm run dev
```

Access at: http://localhost:5173

## 🧪 Testing

### Run All Tests

```bash
# Backend tests
./scripts/test-all.sh

# Individual service tests
cd auth-service && go test ./...
cd location-service && go test ./...
cd route-service && go test ./...
```

### Property-Based Tests

```bash
# Run property tests with 100 iterations
go test -v -tags=property ./...
```

### Integration Tests

```bash
# Requires Docker for testcontainers
go test -v -tags=integration ./...
```

## 📦 Deployment

### Railway Deployment

1. **Create Railway Project**
```bash
railway init
```

2. **Add PostgreSQL (Neon)**
```bash
railway add --plugin neon
```

3. **Add Redis (Upstash)**
```bash
railway add --plugin upstash-redis
```

4. **Set Environment Variables**
```bash
railway variables set JWT_SECRET=$(openssl rand -base64 32)
```

5. **Deploy Services**
```bash
cd api-gateway && railway up
cd auth-service && railway up
cd location-service && railway up
cd route-service && railway up
```

### Docker Deployment

```bash
# Build all services
docker-compose build

# Deploy
docker-compose up -d
```

## 🔑 API Documentation

### Authentication

**POST /auth/login**
```json
{
  "username": "driver1",
  "password": "password123",
  "role": "driver"
}
```

Response:
```json
{
  "token": "eyJhbGc...",
  "expiresAt": "2024-01-15T22:30:00Z",
  "role": "driver",
  "userId": "uuid"
}
```

### Routes

**GET /routes**
```bash
curl http://localhost:8080/routes
```

**POST /routes** (Admin only)
```json
{
  "name": "Route A",
  "stops": [
    {
      "name": "Main Gate",
      "latitude": 23.7104,
      "longitude": 90.4074
    }
  ]
}
```

### Location Updates

**POST /location/update** (Driver only)
```json
{
  "busId": "uuid",
  "latitude": 23.7104,
  "longitude": 90.4074,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**WebSocket /ws/location**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/location');
ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log(update);
};
```

## 🔒 Security

- JWT-based authentication with role-based access control
- Bcrypt password hashing (cost 12)
- Rate limiting on authentication endpoints
- CORS enforcement
- Input validation on all endpoints
- SQL injection prevention with parameterized queries

## 📊 Monitoring

### Health Checks

```bash
# Check all services
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health
```

### Metrics

- Connection pool statistics
- Redis pub/sub metrics
- WebSocket connection counts
- Request latency tracking

## 🐛 Troubleshooting

### Database Connection Issues

```bash
# Check PostgreSQL is running
pg_isready

# Test connection
psql -h localhost -U user -d bus_tracker
```

### Redis Connection Issues

```bash
# Check Redis is running
redis-cli ping

# Monitor Redis
redis-cli monitor
```

### WebSocket Connection Issues

```bash
# Test WebSocket connection
wscat -c ws://localhost:8080/ws/location
```

## 📝 License

MIT License - see LICENSE file for details

## 👥 Contributors

- Your Name - Initial work

## 🙏 Acknowledgments

- Jagannath University Transportation Department
- Go community
- React community
- Android community
