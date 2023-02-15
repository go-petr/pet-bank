// Package configpkg provides parsing functionality for environment variables.
package configpkg

import (
	"time"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
//
// The values are read by viper fron a config file or environement variables.
type Config struct {
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	ServerAddress        string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	Environement         string        `mapstructure:"GO_ENV"`
}

// Load read configuration from file or environment variables.
func Load(path string) (Config, error) {
	var c Config

	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return c, err
	}

	err = viper.Unmarshal(&c)
	if err != nil {
		return c, err
	}

	return c, nil
}
