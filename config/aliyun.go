package config

import "github.com/spf13/viper"

func init() {
	viper.SetDefault("aliyun.region_id", "cn-hangzhou")
	viper.SetDefault("aliyun.access_key_id", "")
	viper.SetDefault("aliyun.access_key_secret", "")
	viper.SetDefault("aliyun.dm.name", "一言网")
	viper.SetDefault("aliyun.dm.mail", "notification@mail.hitokoto.cn")
	viper.SetDefault("aliyun.dm.endpoint", "dm.aliyuncs.com")
}

type SAliyun struct {
}

var aliyun *SAliyun

func Aliyun() *SAliyun {
	if aliyun == nil {
		aliyun = &SAliyun{}
	}
	return aliyun
}

func (t *SAliyun) RegionId() string {
	return viper.GetString("aliyun.region_id")
}

func (t *SAliyun) AccessKeyId() string {
	return viper.GetString("aliyun.access_key_id")
}

func (t *SAliyun) AccessKeySecret() string {
	return viper.GetString("aliyun.access_key_secret")
}

var aliyunDM *dm

func (t *SAliyun) DM() *dm {
	if aliyunDM == nil {
		aliyunDM = &dm{}
	}
	return aliyunDM
}

type dm struct {
}

func (t *dm) AccessKeyId() string {
	if v := viper.GetString("aliyun.dm.access_key_id"); v != "" {
		return v
	} else {
		return viper.GetString("aliyun.access_key_id")
	}
}

func (t *dm) AccessKeySecret() string {
	if v := viper.GetString("aliyun.dm.access_key_secret"); v != "" {
		return v
	} else {
		return viper.GetString("aliyun.access_key_secret")
	}
}

func (t *dm) Endpoint() string {
	return viper.GetString("aliyun.dm.endpoint")
}

func (t *dm) RegionID() string {
	if v := viper.GetString("aliyun.dm.region_id"); v != "" {
		return v
	} else {
		return viper.GetString("aliyun.region_id")
	}
}

func (t *dm) Name() string {
	return viper.GetString("aliyun.dm.name")
}

func (t *dm) Mail() string {
	return viper.GetString("aliyun.dm.mail")
}
