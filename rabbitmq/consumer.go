package rabbitmq

import (
	"context"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	hcontext "source.hitokoto.cn/hitokoto/notification-worker/context"
	"source.hitokoto.cn/hitokoto/notification-worker/logging"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	// Producer UUID
	UUID     string
	RabbitMQ *RabbitMQ
	// The communication channel over connection
	channel *amqp.Channel
	// All deliveries from server will send to this channel
	deliveries <-chan amqp.Delivery
	// This handler will be called when a
	handler func(ctx context.Context, delivery amqp.Delivery) error
	// A notifiyng channel for publishings
	// will be used for sync. between close channel and consume handler
	done chan error
	// This signal is intended to notify close to  shutdown gracefully
	closeSignal chan int
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
func (r *RabbitMQ) NewConsumer(e Exchange, q Queue, bo BindingOptions, co ConsumerOptions) (*Consumer, error) {
	if r.conn == nil {
		return nil, errors.WithStack(errors.New("[RabbitMQ.Consumer] connection is nil"))
	}
	// getting a channel
	mutex.Lock()
	channel, err := r.conn.Channel()
	mutex.Unlock()
	if err != nil {
		return nil, errors.Wrap(err, "[RabbitMQ.Consumer] get channel error")
	}
	uuidInstance, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "[RabbitMQ.Consumer] uuid.NewRandom error")
	}
	c := &Consumer{
		UUID:     uuidInstance.String(),
		RabbitMQ: r,
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
	go func() {
	KeepAliveLoop:
		for {
			select {
			case e := <-c.channel.NotifyClose(make(chan *amqp.Error)):
				c.RabbitMQ.log.Errorf("[RabbitMQ] Channel：%+v \n遇到问题已关闭，错误原因：%v", c.session, e.Error())
				for i := 0; i < 5; i++ {
					c.RabbitMQ.log.Infof("[RabbitMQ] 将在 8 s 后尝试重新注册 Channel: %+v", c.session)
					time.Sleep(8 * time.Second)
					c.RabbitMQ.log.Debugf("[RabbitMQ] 开始注册 Channel：%+v", c.session)
					err := func() error {
						var ch *amqp.Channel
						var err error
						mutex.Lock()
						if ch, err = c.RabbitMQ.Conn().Channel(); err != nil {
							mutex.Unlock()
							return err
						}
						mutex.Unlock()
						c.channel = ch
						return nil
					}()
					if err == nil {
						break KeepAliveLoop
					} else {
						c.RabbitMQ.log.Error("[RabbitMQ] 重注册失败，信息：" + err.Error())
						if i == 4 {
							c.RabbitMQ.log.Fatal("[RabbitMQ] 无法恢复 Channel，进程退出。")
						}
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
func (c *Consumer) Consume(handler func(ctx context.Context, delivery amqp.Delivery) error) error {
	co := c.session.ConsumerOptions
	q := c.session.Queue
	// Exchange bound to Queue, starting Consume
	deliveries, err := c.channel.Consume(
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
	c.deliveries = deliveries
	c.handler = handler

	c.RabbitMQ.log.Infof("[RabbitMQ.Consumer] Tag: %v, deliveries channel started.", c.session.ConsumerOptions.Tag)
	go func() {
		// handle all consumer errors, if required re-connect
		// there are problems with reconnection logic for now
	consumeLoop:
		for {
			select {
			case delivery := <-c.deliveries:
				go func(delivery amqp.Delivery) {
					ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
					defer cancel()
					u4, _ := uuid.NewRandom()
					hctx := hcontext.NewFromContext(ctx)
					logging.NewContext(hctx, zap.String("traceID", u4.String()))
					logging.NewContext(hctx, zap.String("consumerTag", co.Tag))
					logger := logging.WithContext(hctx).Sugar()
					defer logger.Sync()
					go func() {
						if e := handler(hctx, delivery); e != nil {
							logger.Errorf("[RabbitMQ.Consumer] Tag: %v, occurred error: %v, received data: %v", co.Tag, e.Error(), string(delivery.Body))
							if !co.AutoAck && co.AckByError {
								logger.Debug("[RabbitMQ.Consumer] exec NACK")
								if e = delivery.Nack(false, false); e != nil {
									logger.Error(errors.WithMessage(e, "[RabbitMQ.Consumer] ACK Error"))
								}
							}
						} else if !co.AutoAck && co.AckByError {
							if e = delivery.Ack(false); e != nil {
								logger.Error(errors.WithMessage(e, "[RabbitMQ.Consumer] ACK Error"))
							}
						}
					}()

					select {
					case <-ctx.Done():
						logger.Errorf("[RabbitMQ.Consumer] Timeout exceeded. Tag: %v, occurred error: %v, received data: %v", co.Tag, ctx.Err().Error(), string(delivery.Body))
						return
					}
				}(delivery)
			case <-c.closeSignal:
				break consumeLoop
			}
		}
		c.RabbitMQ.log.Info("handle: deliveries channel closed")
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
func (c *Consumer) Get(ctx context.Context, handler func(ctx context.Context, delivery amqp.Delivery) error) error {
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
	defer c.RabbitMQ.log.Sync()
	defer c.RabbitMQ.log.Info("Consumer shutdown OK")
	c.RabbitMQ.log.Info("Waiting for Consumer handler to exit")

	// if we have not called the Consume yet, we can return here
	if c.deliveries == nil {
		close(c.done)
	}

	// this channel is here for finishing the consumer's ranges of
	// delivery chans.  We need every delivery to be processed, here make
	// sure to wait for all consumers goroutines to finish before exiting our
	// process.
	c.closeSignal <- 1
	return <-c.done
}
