package config

import (
	log "github.com/sirupsen/logrus"
	viper "github.com/spf13/viper"
)

func Init() {
	log.Debug("初始化默认配置...")
	loadMain()
	loadRabbitMQ()
	loadAliyun()

	log.Debug("开始读取配置文件...")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")    // 二进制执行目录
	err := viper.ReadInConfig() // 根据以上配置读取加载配置文件
	if err != nil {
		log.Fatal(err) // 读取配置文件失败致命错误
	}
}
