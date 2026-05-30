package main

import (
	"github.com/contracttesting/cli/internal"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	internal.Run()
}
