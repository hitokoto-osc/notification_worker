package consumers

import (
	"github.com/hitokoto-osc/notification-worker/config"
	_ "github.com/hitokoto-osc/notification-worker/consumers/notification/v1"
	"github.com/hitokoto-osc/notification-worker/consumers/provider"
	"github.com/hitokoto-osc/notification-worker/rabbitmq"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func Run() {
	logger := zap.L()
	defer logger.Sync()
	logger.Info("注册消息队列接收器...")
	c := &config.RabbitMQ{}
	logger.Debug((&rabbitmq.Config{
		Host:     c.Host(),
		Port:     c.Port(),
		Username: c.User(),
		Password: c.Pass(),
		Vhost:    c.VHost(),
	}).URI())
	instance := rabbitmq.New(&rabbitmq.Config{
		Host:     c.Host(),
		Port:     c.Port(),
		Username: c.User(),
		Password: c.Pass(),
		Vhost:    c.VHost(),
	}, logger.Sugar())
	if err := instance.Init(); err != nil {
		logger.Fatal("无法启动实例", zap.Error(err))
	}

	// 注册接收器
	options := provider.Get()
	logger.Info("开始注册消息接收器...")
	for _, v := range options {
		instance.RegisterConsumerConfig(*v)
	}
	handleErr(instance.ConsumerSubscribe())
	logger.Info("已注册消息接收器，开始处理消息。")
	signalIntHandler(instance)
	instance.RegisterCloseHandler(func(err error) {
		defer logger.Sync()
		if err != nil {
			logger.Fatal("消息接收器已关闭。", zap.Error(err))
		}
		logger.Warn("消息接收器已关闭。")
	})
}

func handleErr(e error) {
	defer zap.L().Sync()
	if e != nil {
		zap.L().Fatal("无法注册消费者", zap.Error(e))
	}
}

func signalIntHandler(instance *rabbitmq.Instance) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		zap.L().Info("接收到中断信号，开始关闭服务。")
		err := instance.Shutdown()
		if err != nil {
			zap.L().Fatal("关闭服务失败。", zap.Error(err))
		}
		zap.L().Info("优雅退出已完成。")
	}()
}
