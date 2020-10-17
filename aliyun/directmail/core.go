package directmail

import (
	"errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dm"
	log "github.com/sirupsen/logrus"
	config "source.hitokoto.cn/hitokoto/notification-worker/config"
)

var client *dm.Client

func InitAliyunDirectMail() {
	config := &config.Aliyun{}
	var err error
	client, err = dm.NewClientWithAccessKey(config.RegionId(), config.AccessKeyId(), config.AccessKeySecret())
	if err != nil {
		log.Fatalf("无法初始化阿里云邮件推送服务，错误信息： %s \n", err)
	}
}

func SingleSendMail(toAddress string, subject string, body string, isHTML bool) error {
	if client == nil {
		InitAliyunDirectMail()
	}
	request := dm.CreateSingleSendMailRequest()
	config := &config.Aliyun{}
	request.AccountName = config.DM().Mail()
	request.FromAlias = config.DM().Name()
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
		log.Error(response)
		return errors.New("请求状态码不为 200")
	}
	return nil
}
