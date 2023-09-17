package alicloud

import (
	rpc "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dm "github.com/alibabacloud-go/dm-20151123/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/hitokoto-osc/notification-worker/config"
	"github.com/hitokoto-osc/notification-worker/mail/driver"
	"github.com/hitokoto-osc/notification-worker/mail/mailer"
)

var Instance *DM

func init() {
	Instance = NewAliCloudDM()
	driver.Register(driver.TypeAliyun, Instance)
}

type DM struct {
	accessKeyId  string
	accessSecret string
	endpoint     string
	regionID     string

	// Runtime variables
	client  *dm.Client
	options *util.RuntimeOptions
}

func NewAliCloudDM() *DM {
	return &DM{}
}

func (t *DM) Register() (mailer.Sender, error) {
	t.accessKeyId = config.Aliyun().DM().AccessKeyId()
	t.accessSecret = config.Aliyun().DM().AccessKeySecret()
	t.endpoint = config.Aliyun().DM().Endpoint()
	t.regionID = config.Aliyun().RegionId()

	c := new(rpc.Config)
	c.SetAccessKeyId(t.accessKeyId).
		SetAccessKeySecret(t.accessSecret).
		SetEndpoint(t.endpoint).
		SetRegionId(t.regionID)
	client, err := dm.NewClient(c)
	if err != nil {
		return nil, err
	}
	t.client = client
	t.options = new(util.RuntimeOptions).SetAutoretry(true).
		SetMaxAttempts(5).
		SetMaxIdleConns(3)

	return t, nil
}
