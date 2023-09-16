package alicloud

import (
	"context"
	dm "github.com/alibabacloud-go/dm-20151123/v2/client"
	"github.com/cockroachdb/errors"
	"github.com/hitokoto-osc/notification-worker/config"
	"github.com/hitokoto-osc/notification-worker/mail/mailer"
	"strings"
)

func (t *DM) SendSingle(ctx context.Context, m *mailer.Mailer) error {
	switch m.Type {
	case mailer.TypeNormal:
		return t.SendNormalMail(ctx, m)
	case mailer.TypeTemplate:
		return errors.New("阿里云邮件推送服务不支持模板邮件")
	default:
		return errors.New("未知的邮件类型")
	}
}

func (t *DM) SendNormalMail(ctx context.Context, m *mailer.Mailer) error {
	to := m.Mail.To
	if len(to) == 0 {
		return errors.New("收件人不能为空")
	}
	if len(m.Mail.CC) > 0 { // 阿里云不支持抄送
		to = append(to, m.Mail.CC...)
	}
	if len(m.Mail.BCC) > 0 { // 阿里云不支持密送
		to = append(to, m.Mail.BCC...)
	}
	req := new(dm.SingleSendMailRequest)
	req.SetAccountName(config.Aliyun().DM().Mail()).
		SetAddressType(1).
		SetFromAlias(config.Aliyun().DM().Name()).
		SetReplyToAddress(true).
		SetToAddress(strings.Join(to, ",")).
		SetSubject(m.Mail.Subject).
		SetHtmlBody(m.Mail.Body)
	_, err := t.client.SingleSendMailWithOptions(req, t.options)
	if err != nil {
		return errors.Wrap(err, "阿里云邮件推送服务请求失败")
	}
	return nil
}
