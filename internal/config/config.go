package config

import (
	"fmt"

	"github.com/alecthomas/kingpin/v2"
	"github.com/joho/godotenv"
)

type HttpConfig struct {
	Port string
}

type Config struct {
	Debug bool
	Http  HttpConfig
}

func InitConfig() *Config {
	if err := godotenv.Load(".env"); err == nil {
		fmt.Println("loaded environment variables from the .env file")
	}

	config := &Config{}

	kingpin.Flag("debug", "Enable debugging mode.").Default("true").Envar("DEBUG").BoolVar(&config.Debug)
	kingpin.Flag("http.port", "Server port").Default("8080").Envar("HTTP_PORT").StringVar(&config.Http.Port)

	kingpin.HelpFlag.Short('h')
	kingpin.Version("0.0.1")

	return config
}
