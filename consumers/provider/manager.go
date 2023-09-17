package provider

import "github.com/hitokoto-osc/notification-worker/rabbitmq"

var instances []*rabbitmq.ConsumerRegisterOptions = make([]*rabbitmq.ConsumerRegisterOptions, 0, 10)

func Register(config *rabbitmq.ConsumerRegisterOptions) {
	instances = append(instances, config)
}

func Get() []*rabbitmq.ConsumerRegisterOptions {
	return instances
}
