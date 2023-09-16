package alicloud

import (
	dm "github.com/alibabacloud-go/dm-20151123/v2/client"
	rpc "github.com/alibabacloud-go/tea-rpc/client"
	util "github.com/alibabacloud-go/tea-utils/service"
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
	return &DM{
		accessKeyId:  config.Aliyun().DM().AccessKeyId(),
		accessSecret: config.Aliyun().DM().AccessKeySecret(),
		endpoint:     config.Aliyun().DM().Endpoint(),
		regionID:     config.Aliyun().RegionId(),
	}
}

func (t *DM) Register() (mailer.Sender, error) {
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
