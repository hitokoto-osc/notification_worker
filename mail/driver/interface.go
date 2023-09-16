package driver

import "github.com/hitokoto-osc/notification-worker/mail/mailer"

type Type int

const (
	TypeSMTP Type = iota
	TypeSendCloud
	TypeAliyun
	TypeTencentCloud
)

type Driver interface {
	Register() (mailer.Sender, error)
}
