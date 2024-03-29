package config

import (
	"github.com/cockroachdb/errors"
	viper "github.com/spf13/viper"
	"go.uber.org/zap"
	"strings"
)

func Parse() {
	logger := zap.L()
	defer logger.Sync()

	logger.Debug("开始读取配置文件...")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")        // 二进制执行目录
	viper.AddConfigPath("./config") // 二进制执行目录的配置文件夹
	viper.AddConfigPath("./data")
	viper.AddConfigPath("./bin")
	viper.AddConfigPath("..")      // 二进制上级目录
	viper.AddConfigPath("../conf") // 二进制上级目录的配置文件夹
	viper.AddConfigPath(configFile)
	err := viper.ReadInConfig() // 根据以上配置读取加载配置文件
	if err != nil {
		var e viper.ConfigFileNotFoundError
		if !errors.As(err, &e) {
			logger.Fatal("无法解析配置", zap.Error(err)) // 读取配置文件失败致命错误
		}
		logger.Warn("未检测到配置文件，使用环境变量运行。")
	}

	// Parse env config
	viper.SetEnvPrefix("notification_worker")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	logger.Debug("已成功加载配置。",
		zap.String("config_file_used", viper.ConfigFileUsed()),
		zap.Any("settings", viper.AllSettings()),
	)

	executeCallbacks()
}
