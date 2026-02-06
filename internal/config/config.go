package config

import (
	"github.com/Kilat-Pet-Delivery/lib-common/config"
	"github.com/spf13/viper"
)

// StripeConfig holds Stripe-specific configuration.
type StripeConfig struct {
	SecretKey      string
	WebhookSecret  string
}

// ServiceConfig holds all configuration for the payment service.
type ServiceConfig struct {
	Port               string
	AppEnv             string
	DBConfig           config.DatabaseConfig
	JWTConfig          config.JWTConfig
	KafkaConfig        config.KafkaConfig
	StripeConfig       StripeConfig
	PlatformFeePercent float64
}

// Load reads configuration from environment variables and returns a ServiceConfig.
func Load() (*ServiceConfig, error) {
	v, err := config.Load("payment")
	if err != nil {
		return nil, err
	}

	feePercent := v.GetFloat64("PLATFORM_FEE_PERCENT")
	if feePercent <= 0 {
		feePercent = 15.0
	}

	return &ServiceConfig{
		Port:               config.GetServicePort(v, "SERVICE_PORT"),
		AppEnv:             config.GetAppEnv(v),
		DBConfig:           config.LoadDatabaseConfig(v, "DB_NAME"),
		JWTConfig:          config.LoadJWTConfig(v),
		KafkaConfig:        config.LoadKafkaConfig(v),
		StripeConfig:       loadStripeConfig(v),
		PlatformFeePercent: feePercent,
	}, nil
}

// loadStripeConfig extracts Stripe configuration from Viper.
func loadStripeConfig(v *viper.Viper) StripeConfig {
	return StripeConfig{
		SecretKey:     v.GetString("STRIPE_SECRET_KEY"),
		WebhookSecret: v.GetString("STRIPE_WEBHOOK_SECRET"),
	}
}
