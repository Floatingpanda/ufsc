package main

import (
	"log"
	"upforschool/internal/app"
)

func main() {
	log.Println("welcome to upforschool")

	cfg, err := app.ReadConfig("config.json")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	app, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	app.Run()
}
