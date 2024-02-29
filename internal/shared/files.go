package shared

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/traceltrc/pdrive-go-client/internal/shared/constants"
)

func InitConfig() error {
	viper.SetDefault(string(constants.TOKEN), "TOKEN")
	viper.SetDefault(string(constants.API_URL), "API_URL")
	viper.SetDefault(string(constants.CONCURRENT_REQUESTS), 2)

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		msg_err := fmt.Errorf("Unable to get user config directory: %v", err)
		return msg_err
	}

	configDir := filepath.Join(userConfigDir, "pdrive-go-client")
	err = os.MkdirAll(configDir, os.ModePerm)
	if err != nil {
		msg_err := fmt.Errorf("Failed to create config folder: %v", err)
		return msg_err
	}

	viper.AddConfigPath(configDir)
	viper.SetConfigName("config")
	viper.SetConfigType("toml")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			msg_err := fmt.Errorf("Unable to read config: %v", err)
			return msg_err
		} else {
			// Config doesn't exist, create it.
			err := viper.SafeWriteConfig()
			if err != nil {
				msg_err := fmt.Errorf("Unable to write config file: %v", err)
				return msg_err
			}
		}
	}

	return nil
}
