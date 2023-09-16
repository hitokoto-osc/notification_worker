package mail

import (
	"context"
	"github.com/hitokoto-osc/notification-worker/config"
	"github.com/hitokoto-osc/notification-worker/mail/driver"
	mailer "github.com/hitokoto-osc/notification-worker/mail/mailer"
	"go.uber.org/zap"
)

func init() {
	defer zap.L().Sync()
	driverType := driver.Type(config.MailDriver())
	d := driver.Get(driverType)
	if d == nil {
		zap.L().Fatal("无法加载邮件驱动：驱动不存在。", zap.Int("driver", int(driverType)))
	}
	var err error
	instance, err = d.Register()
	if err != nil {
		zap.L().Fatal("无法加载邮件驱动：驱动注册失败。", zap.Error(err), zap.Int("driver", int(driverType)))
	}
}

var instance mailer.Sender

func GetSender() mailer.Sender {
	return instance
}

func SendSingle(ctx context.Context, m *mailer.Mailer) error {
	return instance.SendSingle(ctx, m)
}
