package rabbitmq

import (
	"context"
	"github.com/hitokoto-osc/notification-worker/logging"
	"go.uber.org/zap"
	"strings"
)

type ctx struct {
	context.Context
	m        map[string]any
	instance *Instance
}

type Ctx interface {
	context.Context
	Get(key string) any
	Set(key string, value any)
	Instance() *Instance
	GetProducer(exchangeName, queueName, routingKey string) (*Producer, error)
	GetProducerWithOptions(options ProducerRegisterOptions) (*Producer, error)
	GetProducerByUUID(uuid string) (*Producer, bool)
}

func NewCtxFromContext(c context.Context, instance *Instance) Ctx {
	return &ctx{
		Context:  c,
		m:        make(map[string]any),
		instance: instance,
	}
}

func (c *ctx) Get(key string) any {
	return c.m[key]
}

func (c *ctx) Set(key string, value any) {
	c.m[key] = value
}

func (c *ctx) Instance() *Instance {
	return c.instance
}

func (c *ctx) GetProducer(exchangeName, queueName, routingKey string) (*Producer, error) {
	options := ProducerRegisterOptions{
		Exchange: Exchange{
			Name:    exchangeName,
			Type:    "direct",
			Durable: true,
		},
		Queue: Queue{
			Name:    queueName,
			Durable: true,
		},
		PublishingOptions: PublishingOptions{
			RoutingKey: routingKey,
		},
	}
	return c.GetProducerWithOptions(options)
}

func (c *ctx) GetProducerWithOptions(options ProducerRegisterOptions) (*Producer, error) {
	logger := logging.WithContext(c)
	defer logger.Sync()
	if options.PublishingOptions.RoutingKey == "" {
		options.PublishingOptions.RoutingKey = strings.Trim(options.Exchange.Name+"."+options.Queue.Name, ".")
	}
	return producersCache.getOrCreate(
		options.PublishingOptions.RoutingKey,
		func(uuid string) (*Producer, bool) {
			producer, ok := c.GetProducerByUUID(uuid)
			if !ok {
				logger.Warn("producer not found, try to recreate it.",
					zap.String("uuid", uuid),
					zap.String("routingKey", options.PublishingOptions.RoutingKey),
					zap.String("exchangeName", options.Exchange.Name),
					zap.String("queueName", options.Queue.Name),
				)
			}
			return producer, ok
		},
		func() (*Producer, error) {
			return c.instance.RegisterProducer(options)
		},
	)
}

func (c *ctx) GetProducerByUUID(uuid string) (*Producer, bool) {
	return c.instance.GetProducer(uuid)
}
