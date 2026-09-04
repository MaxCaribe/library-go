package config

import (
	"fmt"

	"github.com/alecthomas/kingpin/v2"
	"github.com/joho/godotenv"
)

type HttpConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL     string
	MinConn int
	MaxConn int
}

type Config struct {
	Debug    bool
	Http     HttpConfig
	Database DatabaseConfig
}

func InitConfig() *Config {
	if err := godotenv.Load(".env"); err == nil {
		fmt.Println("loaded environment variables from the .env file")
	}

	config := &Config{}

	kingpin.Flag("debug", "Enable debugging mode.").Default("true").Envar("DEBUG").BoolVar(&config.Debug)
	kingpin.Flag("http.port", "Server port").Default("8080").Envar("HTTP_PORT").StringVar(&config.Http.Port)

	kingpin.Flag("postgres.url", "Database connection URL").Default("").Envar("DATABASE_URL").StringVar(&config.Database.URL)
	kingpin.Flag("postgres.min-conn", "Idle connections kept in the pool").Default("2").Envar("DATABASE_POOL_MIN_CONNECTIONS").IntVar(&config.Database.MinConn)
	kingpin.Flag("postgres.max-conn", "Maximum open connections").Default("20").Envar("DATABASE_POOL_MAX_CONNECTIONS").IntVar(&config.Database.MaxConn)

	kingpin.HelpFlag.Short('h')
	kingpin.Version("0.0.1")

	return config
}
