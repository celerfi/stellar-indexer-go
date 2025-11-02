package config

import (
	"os"

	"github.com/joho/godotenv"
)


var _ = godotenv.Load("dev.env")


var RPC_URL = os.Getenv("RPC_URL")