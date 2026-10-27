package main

import (
	"log"

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

	application, err := app.New(cfg)

	if err != nil {
		log.Fatal(err)
	}

	defer application.Close()

	if err := application.Run(cfg.Port); err != nil {
		log.Fatal(err)
	}

}
