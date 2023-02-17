package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/armanjr/go-echo-api/db"
	"github.com/armanjr/go-echo-api/handlers"

	"github.com/armanjr/go-echo-api/middlewares"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "Server port")
	dbHost := flag.String("db-host", "db", "Database host")
	dbPort := flag.Int("db-port", 5432, "Database port")
	dbUser := flag.String("db-user", "postgres", "Database user")
	dbPassword := flag.String("db-password", "example", "Database password")
	dbName := flag.String("db-name", "settingsdb", "Database name")
	redisHost := flag.String("redis-host", "redis", "Redis host")
	redisPort := flag.Int("redis-port", 6379, "Redis port")
	secret := flag.String("secret", "my-secret", "JWT secret key")
	flag.Parse()

	// Connect to databases
	pg, err := db.NewPG(*dbHost, *dbPort, *dbUser, *dbPassword, *dbName)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	redis, err := db.NewRedis(*redisHost, *redisPort)
	if err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	// Set up Echo framework
	e := echo.New()
	e.Use(middleware.Logger())

	// Set up request handlers
	h := handlers.NewSettingsHandler(pg, redis, *secret)
	e.GET("/", h.Hello)
	e.POST("/signin", h.SignIn)
	e.GET("/settings", h.GetSettings, middlewares.JWTMiddleware)
	e.GET("/settings/:key", h.GetSetting, middlewares.JWTMiddleware)
	e.POST("/settings", h.CreateSetting, middlewares.JWTMiddleware)
	e.PUT("/settings/:key", h.UpdateSetting, middlewares.JWTMiddleware)

	// Start server
	address := fmt.Sprintf(":%d", *port)
	log.Printf("listening on %s", address)
	if err := e.Start(address); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
