package config

import "github.com/spf13/viper"

func loadMain() {
	viper.SetDefault("debug", true)
}

func Debug() bool {
	return viper.GetBool("debug")
}
