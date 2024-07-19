package main

type Config struct {
	Port        int    `envconfig:"PORT" validate:"required,numeric,min=1,max=65535"`
	LogLevel    string `envconfig:"LOG_LEVEL" validate:"oneof=debug info warn error"`
	DatabaseUrl string `envconfig:"DATABASE_URL" validate:"required,url"`
}
