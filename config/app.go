package config

import "flag"

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
}

func ConfigFile() string {
	return configFile
}

func Debug() bool {
	return debug
}
