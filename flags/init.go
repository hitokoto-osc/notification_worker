package flags

import (
	"flag"
	"fmt"
	"github.com/hitokoto-osc/notification-worker/config"
	"os"
)

var v bool

func init() {
	flag.BoolVar(&v, "v", false, "查看版本信息")
	flag.Parse()
}

func Do() {
	if v {
		fmt.Printf(
			"NotificationWorker ©2023 MoeTeam All Rights Reserved. \n当前版本: %s \n版控哈希: %s\n提交时间：%s\n编译时间：%s\n",
			config.Version,
			config.BuildTag,
			config.CommitTime,
			config.BuildTime,
		)
		os.Exit(0)
	}
}
