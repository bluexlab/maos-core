package main

type Config struct {
	Port             int    `envconfig:"PORT" validate:"required,numeric,min=1,max=65535"`
	DatabaseUrl      string `envconfig:"DATABASE_URL" validate:"omitempty,url"`
	DatabaseHost     string `envconfig:"DATABASE_HOST" validate:"omitempty"`
	DatabasePort     string `envconfig:"DATABASE_PORT" validate:"omitempty"`
	DatabaseUser     string `envconfig:"DATABASE_USER" validate:"omitempty"`
	DatabasePassword string `envconfig:"DATABASE_PASSWORD" validate:"omitempty"`
	DatabaseName     string `envconfig:"DATABASE_NAME" validate:"omitempty"`
	TokenCacheTTL    string `envconfig:"TOKEN_CACHE_TTL" validate:"omitempty"`

	// AWS credentials
	AWSAccessKeyID         string `envconfig:"AWS_ACCESS_KEY_ID" validate:"required"`
	AWSSecretAccessKey     string `envconfig:"AWS_SECRET_ACCESS_KEY" validate:"required"`
	AWSRegion              string `envconfig:"AWS_REGION" validate:"required"`
	SuiteStoreBucket       string `envconfig:"SUITE_STORE_BUCKET" validate:"required"`
	SuiteStorePrefix       string `envconfig:"SUITE_STORE_PREFIX"`
	SuiteStoreScanInterval string `envconfig:"SUITE_STORE_SCAN_INTERVAL" validate:"omitempty"`
	MaosDisplayName        string `envconfig:"MAOS_DISPLAY_NAME" validate:"required"`

	// Azure OpenAI
	AOAIEndpoint string `envconfig:"AOAI_ENDPOINT" validate:"required"`
	AOAIAPIKey   string `envconfig:"AOAI_API_KEY" validate:"required"`

	// Anthropic
	AnthropicAPIKey string `envconfig:"ANTHROPIC_API_KEY" validate:"required"`
}
