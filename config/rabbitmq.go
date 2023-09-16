package config

import "github.com/spf13/viper"

// 加载 RabbitMQ 相关配置

func init() {
	viper.SetDefault("rabbitmq.host", "127.0.0.1")
	viper.SetDefault("rabbitmq.port", 5672)
	viper.SetDefault("rabbitmq.user", "admin")
	viper.SetDefault("rabbitmq.pass", "")
	viper.SetDefault("rabbitmq.vhost", "")
}

type RabbitMQ struct {
}

func (t *RabbitMQ) Host() string {
	return viper.GetString("rabbitmq.host")
}

func (t *RabbitMQ) Port() int {
	return viper.GetInt("rabbitmq.port")
}

func (t *RabbitMQ) User() string {
	return viper.GetString("rabbitmq.user")
}

func (t *RabbitMQ) Pass() string {
	return viper.GetString("rabbitmq.pass")
}

func (t *RabbitMQ) VHost() string {
	return viper.GetString("rabbitmq.vhost")
}
