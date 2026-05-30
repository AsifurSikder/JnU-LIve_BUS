module github.com/university-bus-tracker/auth-service

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.10.9
	github.com/university-bus-tracker/shared v0.0.0
	golang.org/x/crypto v0.17.0
)

replace github.com/university-bus-tracker/shared => ../shared
