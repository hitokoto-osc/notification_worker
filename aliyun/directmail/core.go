package directmail

import (
	"context"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dm"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	config "source.hitokoto.cn/hitokoto/notification-worker/config"
	"source.hitokoto.cn/hitokoto/notification-worker/logging"
)

var client *dm.Client

func InitAliyunDirectMail() {
	logger := logging.GetLogger().Sugar()
	conf := &config.Aliyun{}
	var err error
	client, err = dm.NewClientWithAccessKey(conf.RegionId(), conf.AccessKeyId(), conf.AccessKeySecret())
	if err != nil {
		logger.Fatalf("无法初始化阿里云邮件推送服务，错误信息： %s \n", err)
	}
}

func SingleSendMail(ctx context.Context, toAddress string, subject string, body string, isHTML bool) error {
	logger := logging.WithContext(ctx)
	defer logger.Sync()
	if client == nil {
		InitAliyunDirectMail()
	}
	request := dm.CreateSingleSendMailRequest()
	conf := &config.Aliyun{}
	request.AccountName = conf.DM().Mail()
	request.FromAlias = conf.DM().Name()
	request.AddressType = "1"
	request.ReplyToAddress = "true"
	request.ToAddress = toAddress
	request.Subject = subject
	if isHTML {
		request.HtmlBody = body
	} else {
		request.TextBody = body
	}
	response, err := client.SingleSendMail(request)
	if err != nil {
		return err
	}
	if response.GetHttpStatus() != 200 {
		logger.Error("阿里云邮件推送服务返回状态码不为 200",
			zap.Int("status", response.GetHttpStatus()),
			zap.String("content", response.GetHttpContentString()),
		)

		return errors.New("请求状态码不为 200")
	}
	return nil
}
