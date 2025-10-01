package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func Load() {
	/*
		.env-local для реальной бд, .env.docker для докера
		в скрипте в START пишется или .env-local или .env.docker
		в зависимости от парметров запуска, ./start.sh или ./start.sh docker
	*/
	if err := godotenv.Load(os.Getenv("START")); err != nil {
		log.Fatalf("Env file not found")
	}

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatalf("JWT_SECRET is not set in environment")
	}
	if os.Getenv("MYSQL_DSN") == "" {
		log.Fatalf("MySQLDSN is not set in environment")
	}
	if os.Getenv("MONGO_URI") == "" {
		log.Fatalf("MongoURI is not set in environment")
	}
	if os.Getenv("MONGO_DB_NAME") == "" {
		log.Fatalf("MongoDB is not set in environment")
	}
}
