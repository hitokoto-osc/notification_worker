package mail

type Driver int

const (
	DriverSMTP Driver = iota
	DriverSendCloud
	DriverAliyun
	DriverTencentCloud
)

var instance Sender

func GetSender() Sender {
	return instance
}
