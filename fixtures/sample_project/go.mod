module github.com/vibeguard/team-task-saas

go 1.22

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/go-playground/validator/v10 v10.20.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.5.5
	github.com/nats-io/nats.go v1.35.0
	github.com/ulule/limiter/v3 v3.11.2
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.22.0
	github.com/vibeguard/platform v0.1.0
)

replace github.com/vibeguard/platform => ../../platform