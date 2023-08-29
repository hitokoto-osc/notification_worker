package rabbitmq

import (
	"context"
	"github.com/cockroachdb/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Instance struct {
	RabbitMQ         *RabbitMQ
	Consumers        ConsumerList
	Producers        ProducerList
	consumersOptions []ConsumerRegisterOptions // 用于批量注册
}

// New create rabbitmq wrapper instant
func New(config *Config, logger Logger) *Instance {
	rmq := NewWrapper(config, logger)
	return &Instance{
		RabbitMQ: rmq,
	}
}

// Init initialize instant
func (r *Instance) Init() error {
	if err := r.RabbitMQ.Dial(); err != nil {
		return errors.Wrap(err, "[RabbitMQ] Dial error")
	}
	r.registerChannelRecover()
	return nil
}

// ConsumerRegisterOptions is the options of register consumer
type ConsumerRegisterOptions struct {
	Exchange        Exchange
	Queue           Queue
	BindingOptions  BindingOptions
	ConsumerOptions ConsumerOptions
	CallFunc        func(ctx context.Context, delivery amqp.Delivery) error
}

// RegisterConsumerConfig register a consumer config to queue
func (r *Instance) RegisterConsumerConfig(options ConsumerRegisterOptions) {
	r.consumersOptions = append(r.consumersOptions, options)
}

// ConsumerSubscribe subscribe consumerOptionsList
func (r *Instance) ConsumerSubscribe() error {
	for _, v := range r.consumersOptions {
		err := r.RegisterConsumer(v)
		if err != nil {
			return errors.WithMessagef(err, "consumer Tag: %v", v.ConsumerOptions.Tag)
		}
	}
	return nil
}

// RegisterConsumer register a consumer
func (r *Instance) RegisterConsumer(options ConsumerRegisterOptions) error {
	consumer, err := r.RabbitMQ.NewConsumer(options.Exchange, options.Queue, options.BindingOptions, options.ConsumerOptions)
	if err != nil {
		return err
	}
	err = consumer.Consume(options.CallFunc)
	if err != nil {
		return err
	}
	r.Consumers.Add(ConsumerUnit{
		UUID:     consumer.UUID,
		Consumer: consumer,
	})
	return nil
}

// ProducerRegisterOptions is the options of register producer
type ProducerRegisterOptions struct {
	Exchange          Exchange
	Queue             Queue
	PublishingOptions PublishingOptions
}

// RegisterProducer register a producer
func (r *Instance) RegisterProducer(options ProducerRegisterOptions) (*Producer, error) {
	producer, err := r.RabbitMQ.NewProducer(options.Exchange, options.Queue, options.PublishingOptions)
	if err != nil {
		return nil, err
	}
	r.Producers.Add(ProducerUnit{
		UUID:     producer.UUID,
		Producer: producer,
	})
	return producer, nil
}

// GetConsumer get a exist consumer by uuid
func (r *Instance) GetConsumer(uuid string) (*Consumer, bool) {
	for _, v := range r.Consumers {
		if v.UUID == uuid {
			return v.Consumer, true
		}
	}
	return nil, false
}

// GetProducer get a exist producer by uuid
func (r *Instance) GetProducer(uuid string) (*Producer, bool) {
	for i, v := range r.Producers {
		if v.UUID == uuid {
			if !v.Producer.channel.IsClosed() {
				return v.Producer, true
			}
			// Remove from list
			r.Producers = append(r.Producers[:i], r.Producers[i+1:]...)
		}
	}
	return nil, false
}

// registerChannelRecover is used to recover channel after channel closed
func (r *Instance) registerChannelRecover() {
	go func() {
		for _ = range channelShouldUpdateConn { // ignore data because of notification channel(with useless data)
			r.Consumers.UpdateInstant(r.RabbitMQ)
			r.Producers.UpdateInstant(r.RabbitMQ)
		}
	}()
}

// ConsumerList defines a ConsumerUnit List
type ConsumerList []ConsumerUnit

// ConsumerUnit defines a Consumer unit
type ConsumerUnit struct {
	UUID     string
	Consumer *Consumer
}

// Add a unit to the list
func (p *ConsumerList) Add(unit ConsumerUnit) {
	*p = append(*p, unit)
}

// UpdateInstant update rabbitmq connection(also called instant)
func (p *ConsumerList) UpdateInstant(rmq *RabbitMQ) {
	for _, unit := range *p {
		unit.Consumer.RabbitMQ = rmq
	}
}

// ProducerList defines a ProducerUnit List
type ProducerList []ProducerUnit

// ProducerUnit defines a Producer unit
type ProducerUnit struct {
	UUID     string
	Producer *Producer
}

// Add add a unit to the list
func (p *ProducerList) Add(unit ProducerUnit) {
	*p = append(*p, unit)
}

// UpdateInstant update rabbitmq connection(also called instant)
func (p *ProducerList) UpdateInstant(rmq *RabbitMQ) {
	for _, unit := range *p {
		unit.Producer.RabbitMQ = rmq
	}
}
