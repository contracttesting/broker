package main

import (
	"log"
	"os"

	"github.com/contracttesting/broker/internal"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	components := internal.Run()
	addr := os.Getenv("BROKER_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err := components.Server.Listen(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
