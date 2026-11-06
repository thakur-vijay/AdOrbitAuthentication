package main

// @title AdOrbit Authentication API
// @version 1.0
// @description Authentication service for AdOrbit.
// @host localhost:8081
// @BasePath /api/v1
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your JWT token with Bearer prefix.

import (
	"log"

	swaggerDocs "github.com/adorbit/authentication/docs"
	"github.com/adorbit/authentication/internal/app"
	"github.com/adorbit/authentication/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: .env file not found")
	}

	cfg, err := config.Load()

	if err != nil {
		log.Fatal(err)
	}

	setupSwagger(cfg.SWAGGER_HOST, cfg.Port)
	application, err := app.New(cfg)

	if err != nil {
		log.Fatal(err)
	}

	defer application.Close()

	if err := application.Run(cfg.Port); err != nil {
		log.Fatal(err)
	}

}

func setupSwagger(swaggerHost, authPort string) {
	if swaggerHost == "" {
		swaggerHost = "localhost:8081"
	}

	swaggerDocs.SwaggerInfo.Host = swaggerHost
	swaggerDocs.SwaggerInfo.BasePath = "/api/v1"

	if swaggerHost == "localhost:8081" {
		swaggerDocs.SwaggerInfo.Schemes = []string{"http"}
	} else {
		swaggerDocs.SwaggerInfo.Schemes = []string{"https"}
	}
}
