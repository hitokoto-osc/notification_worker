package config

import (
	"flag"
	"github.com/hitokoto-osc/notification-worker/mail/driver"
	"github.com/spf13/viper"
)

var (
	Version    = "development"
	BuildTag   = "Unknown"
	BuildTime  = "Unknown"
	CommitTime = "Unknown"
	configFile string
	debug      bool
)

func init() {
	flag.StringVar(&configFile, "c", "", "设定配置文件")
	flag.BoolVar(&debug, "d", false, "调试模式")
	viper.SetDefault("mail.driver", driver.TypeAliyun)
}

func ConfigFile() string {
	return configFile
}

func Debug() bool {
	return debug
}

func MailDriver() int {
	return viper.GetInt("mail.driver")
}
