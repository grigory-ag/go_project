package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DB       DBConfig
	Server   ServerConfig
	RabbitMQ RabbitMQConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type ServerConfig struct {
	Port string
}

type RabbitMQConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Vhost    string
}

type WorkersConfig struct {
	RabbitMQ RabbitMQConfig
	Workers  int
}

func New() (*Config, error) {
	rabbitConfig, err := loadRabbitMQConfig()
	if err != nil {
		return nil, err
	}

	return &Config{
		DB: DBConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
		Server: ServerConfig{
			Port: os.Getenv("SERVER_PORT"),
		},
		RabbitMQ: rabbitConfig,
	}, nil
}

func LoadWorkersConfig() (*WorkersConfig, error) {
	rabbitConfig, err := loadRabbitMQConfig()
	if err != nil {
		return nil, err
	}

	workers, err := envInt("WORKERS", 4)
	if err != nil {
		return nil, err
	}
	if workers < 1 {
		return nil, fmt.Errorf("WORKERS must be greater than zero")
	}

	return &WorkersConfig{
		RabbitMQ: rabbitConfig,
		Workers:  workers,
	}, nil
}

func loadRabbitMQConfig() (RabbitMQConfig, error) {
	port, err := envInt("RABBITMQ_PORT", 5672)
	if err != nil {
		return RabbitMQConfig{}, err
	}

	vhost := os.Getenv("RABBITMQ_VHOST")
	if vhost == "" {
		vhost = "/"
	}

	return RabbitMQConfig{
		Host:     os.Getenv("RABBITMQ_HOST"),
		Port:     port,
		User:     os.Getenv("RABBITMQ_USER"),
		Password: os.Getenv("RABBITMQ_PASSWORD"),
		Vhost:    vhost,
	}, nil
}

func envInt(name string, defaultValue int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return parsed, nil
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}
