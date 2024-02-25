package shared

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/traceltrc/pdrive-go-client/internal/shared/utils"
)

type ConfigKey string

const (
  TOKEN ConfigKey = "token"
  API_URL ConfigKey = "api_url"
  CONCURRENT_REQUESTS ConfigKey = "concurrent_requests"
)

func InitConfig() {
  viper.SetDefault(string(TOKEN), "TOKEN")
  viper.SetDefault(string(API_URL), "API_URL")
  viper.SetDefault(string(CONCURRENT_REQUESTS), 2)

  userConfigDir, err := os.UserConfigDir()
  if err != nil {
    utils.ErrorExit("Unable to get user config directory: %v", err)
  }

  configDir := filepath.Join(userConfigDir, "pdrive-go-client")
  err = os.MkdirAll(configDir, os.ModePerm)
  if err != nil {
    utils.ErrorExit("Failed to create config folder: %v", err)
  }

  viper.AddConfigPath(configDir)
  viper.SetConfigName("config")
  viper.SetConfigType("toml")
  
  if err := viper.ReadInConfig(); err != nil {
    if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
      utils.ErrorExit("Unable to read config: %v", err)
    } else {
      // Config doesn't exist, create it.
      err := viper.SafeWriteConfig()
      if err != nil {
        utils.ErrorExit("Unable to write config file: %v", err)
      }
    }
  }
}

