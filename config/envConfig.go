package config

import (
	"os"

	"github.com/joho/godotenv"
)

var _ = godotenv.Load("dev.env")

// db variables
var (
	DB_USER     = os.Getenv("DB_USER")
	DB_PASSWORD = os.Getenv("DB_PASSWORD")
	DB_HOST     = os.Getenv("DB_HOST")
	DB_NAME     = os.Getenv("DB_NAME")
)

var RPC_URL = os.Getenv("RPC_URL")
var DEPLOYMENT_ENVIRONMENT = os.Getenv("DEPLOYMENT_ENVIRONMENT")
