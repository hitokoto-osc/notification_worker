package rabbitmq

import (
	"context"
	"github.com/cockroachdb/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

type Instance struct {
	RabbitMQ         *RabbitMQ
	Consumers        ConsumerList
	Producers        ProducerList
	consumersOptions []ConsumerRegisterOptions // 用于批量注册
	closed           chan bool
	isClosed         bool
	err              error
}

// New create rabbitmq wrapper instant
func New(config *Config, logger Logger) *Instance {
	rmq := NewWrapper(config, logger)
	return &Instance{
		RabbitMQ: rmq,
		closed:   make(chan bool, 1),
	}
}

// Init initialize instance
func (r *Instance) Init() error {
	if err := r.RabbitMQ.Dial(); err != nil {
		return errors.Wrap(err, "[RabbitMQ] Dial error")
	}
	r.registerChannelRecover()
	return nil
}

// Shutdown gracefully close instance
func (r *Instance) Shutdown() error {
	defer func() {
		r.closed <- true
	}()
	r.RabbitMQ.log.Debug("shutdown consumers...")
	r.err = r.Consumers.Shutdown()
	if r.err != nil {
		return errors.Wrap(r.err, "consumers shutdown error")
	}
	r.RabbitMQ.log.Debug("shutdown producers...")
	r.err = r.Producers.Shutdown()
	if r.err != nil {
		return errors.Wrap(r.err, "producers shutdown error")
	}
	r.RabbitMQ.log.Debug("shutdown rabbitmq...")
	r.err = r.RabbitMQ.Shutdown()
	if r.err != nil {
		return errors.Wrap(r.err, "rabbitmq shutdown error")
	}
	return nil
}

// RegisterCloseHandler register a close handler
func (r *Instance) RegisterCloseHandler(fn func(err error)) {
	<-r.closed
	fn(r.err)
	close(r.closed)
	r.isClosed = true
}

func (r *Instance) IsClosed() bool {
	return r.isClosed
}

// ConsumerRegisterOptions is the options of register consumer
type ConsumerRegisterOptions struct {
	Exchange        Exchange
	Queue           Queue
	BindingOptions  BindingOptions
	ConsumerOptions ConsumerOptions
	CallFunc        func(ctx Ctx, delivery amqp.Delivery) error
}

// RegisterConsumerConfig register a consumer config to queue
func (r *Instance) RegisterConsumerConfig(options ConsumerRegisterOptions) {
	r.consumersOptions = append(r.consumersOptions, options)
}

// ConsumerSubscribe subscribe consumerOptionsList
func (r *Instance) ConsumerSubscribe() error {
	return r.ConsumerSubscribeWithTimeout(5 * time.Minute) // 默认超时时间为 5 分钟
}

func (r *Instance) ConsumerSubscribeWithTimeout(timeout time.Duration) error {
	c := context.Background()
	if timeout > 0 { // 如果超时时间大于 0，则使用超时上下文
		var cancel context.CancelFunc
		c, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}
	eg, _ := errgroup.WithContext(c)
	for _, v := range r.consumersOptions {
		co := v // copy
		eg.Go(func() error {
			err := r.RegisterConsumer(co)
			if err != nil {
				return errors.WithMessagef(err, "consumer Tag: %v", co.ConsumerOptions.Tag)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return errors.Wrap(err, "consumer subscribe error")
	}
	return nil
}

// RegisterConsumer register a consumer
func (r *Instance) RegisterConsumer(options ConsumerRegisterOptions) error {
	var consumer *Consumer
	var e = make(chan error)
	go func() {
		var err error
		consumer, err = r.RabbitMQ.NewConsumer(r, options.Exchange, options.Queue, options.BindingOptions, options.ConsumerOptions)
		if err != nil {
			e <- err
			return
		}
		e <- consumer.Consume(options.CallFunc)
	}()
	if err := <-e; err != nil {
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
	producer, err := r.RabbitMQ.NewProducer(r, options.Exchange, options.Queue, options.PublishingOptions)
	if err != nil {
		return nil, err
	}
	r.Producers.Add(ProducerUnit{
		UUID:     producer.UUID,
		Producer: producer,
	})
	return producer, nil
}

// GetConsumer get an exist consumer by uuid
func (r *Instance) GetConsumer(uuid string) (*Consumer, bool) {
	return r.Consumers.Get(uuid)
}

// GetProducer get an exist producer by uuid
func (r *Instance) GetProducer(uuid string) (*Producer, bool) {
	return r.Producers.Get(uuid)
}

// registerChannelRecover is used to recover channel after channel closed
func (r *Instance) registerChannelRecover() {
	go func() {
		for _ = range channelShouldUpdateConn { // ignore data because of notification channel(with useless data)
			r.Consumers.UpdateInstance(r.RabbitMQ)
			r.Producers.UpdateInstance(r.RabbitMQ)
		}
	}()
}

// ConsumerList defines a ConsumerUnit List
type ConsumerList struct {
	list []ConsumerUnit
	sync.RWMutex
}

// ConsumerUnit defines a Consumer unit
type ConsumerUnit struct {
	UUID     string
	Consumer *Consumer
}

// Add a unit to the list
func (p *ConsumerList) Add(unit ConsumerUnit) {
	p.Lock()
	defer p.Unlock()
	p.list = append(p.list, unit)
}

// UpdateInstance update rabbitmq connection(also called instant)
func (p *ConsumerList) UpdateInstance(rmq *RabbitMQ) {
	p.Lock()
	defer p.Unlock()
	for _, unit := range p.list {
		unit.Consumer.RabbitMQ = rmq
	}
}

func (p *ConsumerList) Get(uuid string) (*Consumer, bool) {
	p.RLock()
	defer p.RUnlock()
	for _, v := range p.list {
		if v.UUID == uuid {
			return v.Consumer, true
		}
	}
	return nil, false
}

func (p *ConsumerList) Remove(uuid string) {
	p.Lock()
	defer p.Unlock()
	for i, v := range p.list {
		if v.UUID == uuid {
			p.list = append(p.list[:i], p.list[i+1:]...)
		}
	}
}

func (p *ConsumerList) Shutdown() error {
	p.Lock()
	defer p.Unlock()
	for _, unit := range p.list {
		if err := unit.Consumer.Shutdown(); err != nil {
			return err
		}
	}
	p.list = nil
	return nil
}

// ProducerList defines a ProducerUnit List
type ProducerList struct {
	list []ProducerUnit
	sync.RWMutex
}

// ProducerUnit defines a Producer unit
type ProducerUnit struct {
	UUID     string
	Producer *Producer
}

// Add a unit to the list
func (p *ProducerList) Add(unit ProducerUnit) {
	p.Lock()
	defer p.Unlock()
	p.list = append(p.list, unit)
}

// UpdateInstance update rabbitmq connection(also called instant)
func (p *ProducerList) UpdateInstance(rmq *RabbitMQ) {
	p.Lock()
	defer p.Unlock()
	for _, unit := range p.list {
		unit.Producer.RabbitMQ = rmq
	}
}

func (p *ProducerList) Get(uuid string) (*Producer, bool) {
	p.RLock()
	defer p.RUnlock()
	for _, v := range p.list {
		if v.UUID == uuid {
			if !v.Producer.channel.IsClosed() { // 如果通道未关闭，则返回
				return v.Producer, true
			}
			p.Remove(uuid)
		}
	}
	return nil, false
}

func (p *ProducerList) Remove(uuid string) {
	p.Lock()
	defer p.Unlock()
	for i, v := range p.list {
		if v.UUID == uuid {
			p.list = append(p.list[:i], p.list[i+1:]...)
		}
	}
}

func (p *ProducerList) Shutdown() error {
	p.Lock()
	defer p.Unlock()
	for _, unit := range p.list {
		if err := unit.Producer.Shutdown(); err != nil {
			return err
		}
	}
	p.list = nil
	return nil
}
