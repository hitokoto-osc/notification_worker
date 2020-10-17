package config

import "github.com/spf13/viper"

func loadAliyun() {
	viper.SetDefault("aliyun.region_id", "cn-hangzhou")
	viper.SetDefault("aliyun.access_key_id", "")
	viper.SetDefault("aliyun.access_key_secret", "")
	viper.SetDefault("aliyun.dm.name", "一言网")
	viper.SetDefault("aliyun.dm.mail", "notification@mail.hitokoto.cn")
}

type Aliyun struct {
}

func (t *Aliyun) RegionId() string {
	return viper.GetString("aliyun.region_id")
}

func (t *Aliyun) AccessKeyId() string {
	return viper.GetString("aliyun.access_key_id")
}

func (t *Aliyun) AccessKeySecret() string {
	return viper.GetString("aliyun.access_key_secret")
}

func (t *Aliyun) DM() *dm {
	return &dm{}
}

type dm struct {
}

func (t *dm) Name() string {
	return viper.GetString("aliyun.dm.name")
}

func (t *dm) Mail() string {
	return viper.GetString("aliyun.dm.mail")
}
