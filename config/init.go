package config

import (
	viper "github.com/spf13/viper"
	"go.uber.org/zap"
	"source.hitokoto.cn/hitokoto/notification-worker/logging"
)

func Init(path string) {
	logger := logging.GetLogger()
	logger.Debug("初始化默认配置...")
	loadMain()
	loadRabbitMQ()
	loadAliyun()

	logger.Debug("开始读取配置文件...")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".") // 二进制执行目录
	viper.AddConfigPath("./bin")
	viper.AddConfigPath("..")      // 二进制上级目录
	viper.AddConfigPath("../conf") // 二进制上级目录的配置文件夹
	viper.AddConfigPath(path)
	err := viper.ReadInConfig() // 根据以上配置读取加载配置文件
	if err != nil {
		logger.Fatal("无法解析配置", zap.Error(err)) // 读取配置文件失败致命错误
	}
	logger.Debug("使用配置文件：", zap.String("configFileUsed", viper.ConfigFileUsed()))
}
