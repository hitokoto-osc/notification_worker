package rabbitmq

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/hitokoto-osc/notification-worker/logging"
	"go.uber.org/zap"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	// Producer UUID
	UUID     string
	RabbitMQ *RabbitMQ
	// The instance that this producer belongs to
	instance *Instance
	// The communication channel over connection
	channel *amqp.Channel
	// All deliveries from server will send to this channel
	deliveries <-chan amqp.Delivery
	// This handler will be called when a
	handler func(ctx Ctx, delivery amqp.Delivery) error
	// A notifying channel for publishing's
	// will be used for sync. between close channel and consume handler
	done chan error
	// Current producer connection settings
	session Session
}

type ConsumerOptions struct {
	// The consumer is identified by a string that is unique and scoped for all
	// consumers on this channel.
	Tag string
	// When autoAck (also known as noAck) is true, the server will acknowledge
	// deliveries to this consumer prior to writing the delivery to the network.  When
	// autoAck is true, the consumer should not call Delivery.Ack
	AutoAck    bool // autoAck
	AckByError bool // ack by system(if consumer func return error)
	// Check Queue struct documentation
	Exclusive bool // exclusive
	// When noLocal is true, the server will not deliver publishing sent from the same
	// connection to this consumer. (Do not use Publish and Consume from same channel)
	NoLocal bool // noLocal
	// Check Queue struct documentation
	NoWait bool // noWait
	// Check Exchange comments for Args
	Args amqp.Table // arguments
}

func (c *Consumer) Deliveries() <-chan amqp.Delivery {
	return c.deliveries
}

// NewConsumer is a constructor for consumer creation
// Accepts Exchange, Queue, BindingOptions and ConsumerOptions
func (r *RabbitMQ) NewConsumer(instance *Instance, e Exchange, q Queue, bo BindingOptions, co ConsumerOptions) (*Consumer, error) {
	if r.conn == nil {
		return nil, errors.WithStack(errors.New("[RabbitMQ.Consumer] connection is nil"))
	}
	// getting a channel
	channel, err := r.conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "[RabbitMQ.Consumer] get channel error")
	}
	c := &Consumer{
		UUID:     uuid.NewString(),
		RabbitMQ: r,
		instance: instance,
		channel:  channel,
		done:     make(chan error),
		session: Session{
			Exchange:        e,
			Queue:           q,
			ConsumerOptions: &co,
			BindingOptions:  bo,
		},
	}
	c.HandleError() // handle Error
	err = c.Register()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// HandleError register the recover loop against channel closed
func (c *Consumer) HandleError() {
	logger := c.RabbitMQ.log
	go func() {
		for e := range c.channel.NotifyClose(make(chan *amqp.Error)) {
			if e == nil { // channel closed by user
				logger.Infof("[RabbitMQ] 优雅退出：Channel %s 保活方法已停止。", c.session.ConsumerOptions.Tag)
				break
			}
			logger.Errorf("[RabbitMQ] Channel：%+v \n 遇到问题已关闭，错误原因：%v", c.session, e.Error())
			for i := 0; i < 5; i++ {
				logger.Infof("[RabbitMQ] 将在 8 s 后尝试重新注册 Channel: %+v", c.session)
				time.Sleep(8 * time.Second)
				logger.Debugf("[RabbitMQ] 开始注册 Channel：%+v", c.session)
				err := func() error {
					var ch *amqp.Channel
					var err error
					if ch, err = c.RabbitMQ.Conn().Channel(); err != nil {
						return err
					}
					c.channel = ch
					return nil
				}()
				if err == nil {
					break
				} else {
					logger.Error("[RabbitMQ] 重注册失败，信息：" + err.Error())
					if i == 4 {
						logger.Fatal("[RabbitMQ] 无法恢复 Channel，进程退出。")
					}
				}

			}
		}
	}()
}

// Register connect internally declares the exchanges and queues
func (c *Consumer) Register() error {
	e := c.session.Exchange
	q := c.session.Queue
	bo := c.session.BindingOptions

	var err error

	// declaring Exchange
	if err = c.channel.ExchangeDeclare(
		e.Name,       // name of the exchange
		e.Type,       // type
		e.Durable,    // durable
		e.AutoDelete, // delete when complete
		e.Internal,   // internal
		e.NoWait,     // noWait
		e.Args,       // arguments
	); err != nil {
		return err
	}

	// declaring Queue
	_, err = c.channel.QueueDeclare(
		q.Name,       // name of the queue
		q.Durable,    // durable
		q.AutoDelete, // delete when usused
		q.Exclusive,  // exclusive
		q.NoWait,     // noWait
		q.Args,       // arguments
	)
	if err != nil {
		return err
	}

	// binding Exchange to Queue
	if err = c.channel.QueueBind(
		// bind to real queue
		q.Name,        // name of the queue
		bo.RoutingKey, // bindingKey
		e.Name,        // sourceExchange
		bo.NoWait,     // noWait
		bo.Args,       // arguments
	); err != nil {
		return err
	}
	return nil
}

// Consume accepts a handler function for every message streamed from RabbitMq
// will be called within this handler func
func (c *Consumer) Consume(handler func(ctx Ctx, delivery amqp.Delivery) error) error {
	co := c.session.ConsumerOptions
	q := c.session.Queue
	// Exchange bound to Queue, starting Consume
	var err error
	c.deliveries, err = c.channel.Consume(
		//ass consume from real queue
		q.Name,       // name
		co.Tag,       // consumerTag,
		co.AutoAck,   // autoAck
		co.Exclusive, // exclusive
		co.NoLocal,   // noLocal
		co.NoWait,    // noWait
		co.Args,      // arguments
	)
	if err != nil {
		return err
	}

	// should we stop streaming, in order not to consume from server?
	c.handler = handler

	logger := c.RabbitMQ.log
	logger.Infof("[RabbitMQ.Consumer] Tag: %v, deliveries channel started.", c.session.ConsumerOptions.Tag)
	go func() {
		for delivery := range c.deliveries {
			go func(delivery amqp.Delivery) {
				ctxWithDeadline, cancel := context.WithTimeout(context.Background(), time.Hour) // 设置为 1 小时执行超时（照顾失败策略）
				defer cancel()
				u := uuid.NewString()
				rCtx := NewCtxFromContext(ctxWithDeadline, c.instance)
				logging.NewContext(rCtx,
					zap.String("trace_id", u),
					zap.String("consumer_tag", co.Tag),
				)
				log := logging.WithContext(rCtx)
				defer log.Sync()
				done := make(chan bool)
				go func() {
					defer func() {
						done <- true
					}()
					if e := c.handler(rCtx, delivery); e != nil {
						log.Error(
							"[RabbitMQ.Consumer] Occur a error while consuming a message. ",
							zap.Error(e),
							zap.Any("headers", delivery.Headers),
							zap.ByteString(
								"body",
								delivery.Body,
							),
						)
						if !co.AutoAck && co.AckByError {
							log.Debug("[RabbitMQ.Consumer] exec NACK")
							if e = delivery.Nack(false, false); e != nil {
								log.Error(
									"ACK failed:",
									zap.Error(errors.WithMessage(e, "[RabbitMQ.Consumer] ACK Error")),
								)
							}
						}
					} else if !co.AutoAck && co.AckByError {
						if e = delivery.Ack(false); e != nil {
							log.Error(
								"ACK failed:",
								zap.Error(errors.WithMessage(e, "[RabbitMQ.Consumer] ACK Error")),
							)
						}
					}
				}()

				select {
				case <-rCtx.Done():
					log.Error(
						"[RabbitMQ.Consumer] Timeout exceeded.",
						zap.Error(rCtx.Err()),
						zap.ByteString("body", delivery.Body),
					)
					return
				case <-done:
					log.Debug("[RabbitMQ.Consumer] done")
				}
			}(delivery)
		}

		logger.Info("handle: deliveries channel closed")
		c.done <- nil
	}()
	return nil
}

// QOS controls how many messages the server will try to keep on the network for
// consumers before receiving delivery acks.  The intent of Qos is to make sure
// the network buffers stay full between the server and client.
func (c *Consumer) QOS(messageCount int) error {
	return c.channel.Qos(messageCount, 0, false)
}

// Get ConsumeMessage accepts a handler function and only consumes one message
// stream from RabbitMq
func (c *Consumer) Get(ctx Ctx, handler func(ctx Ctx, delivery amqp.Delivery) error) error {
	defer c.RabbitMQ.log.Sync()
	co := c.session.ConsumerOptions
	q := c.session.Queue
	message, ok, err := c.channel.Get(q.Name, co.AutoAck)
	if err != nil {
		return err
	}

	c.handler = handler
	if ok {
		c.RabbitMQ.log.Debug("Message received")
		err = handler(ctx, message)
		if err != nil {
			return err
		}
	} else {
		c.RabbitMQ.log.Debug("No message received")
	}

	// TODO maybe we should return ok too?
	return nil
}

// Shutdown gracefully closes all connections and waits
// for handler to finish its messages
func (c *Consumer) Shutdown() error {
	co := c.session.ConsumerOptions
	if err := shutdownChannel(c.channel, co.Tag); err != nil {
		return err
	}
	logger := c.RabbitMQ.log
	defer logger.Sync()
	defer logger.Infof("Consumer %s shutdown OK", co.Tag)
	logger.Infof("Waiting for Consumer %s handler to exit", co.Tag)

	// if we have not called the Consume yet, we can return here
	if c.deliveries == nil {
		close(c.done)
	}

	// this channel is here for finishing the consumer's ranges of
	// delivery chans.  We need every delivery to be processed, here make
	// sure to wait for all consumers goroutines to finish before exiting our
	return <-c.done
}
