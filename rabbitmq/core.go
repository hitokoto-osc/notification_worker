package rabbitmq

import (
	// "context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"gopkg.in/errgo.v2/errors"
	"runtime"
	"time"
)

// 解耦来源：
// 简书: https://juejin.im/post/5d39101bf265da1bd3059da6
// Github: https://github.com/callistaenterprise/goblog/blob/99035a1b3d8f38e1b6577f5bfc2357a4d2be9a6d/vipservice/cmd/vipservice/main.go
// 由于经过深度改造，所以此版由我维护。不得不吐槽，以上两位作者写的都是一坨 X，而且 Go 现有轮子太少了。

// TODO: 修复 PublishOnQueueWithContext（移除 tracing 的基础上）

// 定义接口
type Client interface {
	ConnectToBroker(connectionString string)
	Publish(body []byte, exchangeName string, exchangeType string, bindingKey string) error
	// PublishOnQueue(msg []byte, queueName string) error
	PublishOnQueue([]byte, string, string, string) error
	PublishOnQueueWithContext(body []byte, queueName string, exchangeName string, routingKey string) error
	// PublishOnQueueWithContext(ctx context.Context, msg []byte, queueName string) error
	// Subscribe(exchangeName string, exchangeType string, consumerName string, handlerFunc func(amqp.Delivery)) error
	Subscribe(exchangeName string, exchangeType string, queueName string, consumerName string, bindingKey string, handlerFunc func(amqp.Delivery) error) error
	SubscribeToQueue(queueName string, consumerName string, handlerFunc func(amqp.Delivery) error) error
	Close()
}

// 定义接受者，使其与客户端解耦
type Receiver struct {
	// 接收者的各项信息
	ExchangeName string
	ExchangeType string
	QueueName    string
	BindingKey   string
	ConsumerName string
	Deliveries   chan amqp.Delivery
	HandlerFunc  func(msg amqp.Delivery) error //定义一个处理方法
}

// 定义消费端,消费端持有调用端和接收者
type Consumer struct {
	Client    Client //一个客户端
	Receivers []*Receiver
}

func (c *Consumer) Add(rec ...*Receiver) {
	// 添加接收器
	c.Receivers = append(c.Receivers, rec...)

}

// 订阅接收器到交换器
func (c *Consumer) Subscribe() {
	for _, receiver := range c.Receivers {
		err := c.Client.Subscribe(
			receiver.ExchangeName,
			receiver.ExchangeType,
			receiver.QueueName,
			receiver.ConsumerName,
			receiver.BindingKey,
			receiver.HandlerFunc,
		)
		if err != nil {
			log.Printf("处理 Subscribe 时发生错误: %v   %s ", receiver, err)
		}
	}
}

// 订阅接收器到特定队列
func (c *Consumer) SubscribeToQueue() {
	for _, receiver := range c.Receivers {
		err := c.Client.SubscribeToQueue(receiver.QueueName,
			receiver.ConsumerName,
			receiver.HandlerFunc)
		if err != nil {
			log.Printf("SubscribeToQueue error: %v   %s ", receiver, err)
		}
	}

}

// AmqpClient is our real implementation, encapsulates a pointer to an amqp.Connection
type AmqpClient struct {
	conn *amqp.Connection
}

// ConnectToBroker connects to an AMQP broker using the supplied connectionString.
func (m *AmqpClient) ConnectToBroker(connectionString string) {
	if connectionString == "" {
		panic("Cannot initialize connection to broker, connectionString not set. Have you initialized?")
	}

	var err error
	conn, err := amqp.Dial(fmt.Sprintf("%s/", connectionString))
	if err != nil {
		panic("Failed to connect to AMQP compatible broker at: " + connectionString)
	}
	m.conn = conn
	go func() {
	KeepAliveLoop:
		for {
			select {
			case err := <-conn.NotifyClose(make(chan *amqp.Error)):
				log.Error("[RabbitMQ] AMQP连接丢失，错误信息" + err.Error())
				for i := 0; i < 5; i++ {
					log.Info("[RabbitMQ] AMQP 连接将在 5 秒后尝试重连接...")
					time.Sleep(5 * time.Second)
					e := func() (errTmp error) {
						defer func() {
							if e := recover(); e != nil {
								switch e := e.(type) {
								case string:
									errTmp = errors.New(e)
								case error:
									errTmp = e
								case runtime.Error:
									errTmp = e
								default:
									panic(e)
								}
							}
						}()
						m.ConnectToBroker(connectionString)
						return
					}()
					if e != nil {
						log.Error("[RabbitMQ] AMQP 重连接失败，错误信息：" + e.Error())
					} else {
						log.Info("[RabbitMQ] AMQP 连接已成功重建")
						break KeepAliveLoop
					}
					if i == 4 {
						log.Fatal("[RabbitMQ] AMQP 重连接次数过多，程序退出。")
					}

				}
			}
		}
	}()
}

// Publish publishes a message to the named exchange.
func (m *AmqpClient) Publish(body []byte, exchangeName string, exchangeType string, bindingKey string) error {
	if m.conn == nil {
		log.Fatal("RabbitMQ 连接还未初始化， 请不要在连接初始化前尝试提交 Publish")
	}
	ch, err := m.conn.Channel() // Get a channel from the connection
	if err != nil {
		failOnError(err, "无法获得 RabbitMQ Channel")
	}
	defer func() { // 关闭 Channel 连接
		err = ch.Close()
		if err != nil {
			failOnError(err, "无法关闭 Channel 连接")
		}
	}()
	err = ch.ExchangeDeclare(
		exchangeName, // exchange 名字
		exchangeType, // 种类
		true,         // 是否持久化
		false,        // 完成时删除
		false,        // internal
		false,        // noWait
		nil,          // arguments
	)
	failOnError(err, "无法注册 Exchange")

	queue, err := ch.QueueDeclare( // Declare a queue that will be created if not exists with some args
		"",    // our queue name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}
	err = ch.QueueBind(
		queue.Name,   // name of the queue
		bindingKey,   // bindingKey
		exchangeName, // sourceExchange
		false,        // noWait
		nil,          // arguments
	)

	if err != nil {
		return err
	}

	err = ch.Publish( // Publishes a message onto the queue.
		exchangeName, // exchange
		bindingKey,   // routing key      q.Name
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			Body: body, // Our JSON body as []byte
		})
	log.Infof("A message was sent: %v", string(body))
	return err
}

// TODO: 搞清楚这个沙雕玩意到底要干啥
// PublishOnQueueWithContext publishes the supplied body onto the named queue, passing the context.
func (m *AmqpClient) PublishOnQueueWithContext(body []byte, queueName string, exchangeName string, routingKey string) error {
	if m.conn == nil {
		panic("Tried to send message before connection was initialized. Don't do that.")
	}
	ch, err := m.conn.Channel() // 从连接中获得 Channel
	if err != nil {
		failOnError(err, "无法从连接池中获得 Channel")
	}
	defer func() {
		err = ch.Close()
		if err != nil {
			failOnError(err, "无法关闭 Channel")
		}
	}()

	// queue, err := ch.QueueDeclare( // Declare a queue that will be created if not exists with some args
	//	queueName, // our queue name
	//	true,     // durable
	//	false,     // delete when unused
	//	false,     // exclusive
	//	false,     // no-wait
	//	nil,       // arguments
	// )
	failOnError(err, "无法从 Channel 中获得 queue 信息")
	// Publishes a message onto the queue.
	err = ch.Publish(
		exchangeName, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		buildMessage(body))
	log.Debugf("A message was sent to queue %v: %v", queueName, string(body))
	return err
}

func buildMessage(body []byte) amqp.Publishing {
	publishing := amqp.Publishing{
		ContentType: "application/json",
		Body:        body, // Our JSON body as []byte
	}
	return publishing
}

// PublishOnQueue publishes the supplied body on the queueName.
func (m *AmqpClient) PublishOnQueue(body []byte, queueName string, exchangeName string, routingKey string) error {
	return m.PublishOnQueueWithContext(body, queueName, exchangeName, routingKey)
}

// Subscribe registers a handler function for a given exchange.
func (m *AmqpClient) Subscribe(exchangeName string, exchangeType string, queueName string, consumerName string, bindingKey string, handlerFunc func(amqp.Delivery) error) error {
	ch, err := m.conn.Channel()
	failOnError(err, "Failed to open a channel")
	// defer ch.Close()

	err = ch.ExchangeDeclare(
		exchangeName, // name of the exchange
		exchangeType, // type
		true,         // durable
		false,        // delete when complete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	)

	failOnError(err, "Subscribe: 注册新 Exchange 失败")

	log.Printf("Subscribe: 已注册 Exchange: %s, 开始注册 Queue: (%s)", exchangeName, queueName)
	queue, err := ch.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)
	failOnError(err, "Subscribe: 无法注册 Queue")

	log.Printf("成功注册 (%d messages, %d consumers), 绑定到 Exchange (键 '%s')",
		queue.Messages, queue.Consumers, exchangeName)

	err = ch.QueueBind(
		queueName,    // name of the queue
		bindingKey,   // bindingKey
		exchangeName, // sourceExchange
		false,        // noWait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("Queue Bind: %s \n", err)
	}

	msgs, err := ch.Consume(
		queueName,    // queue
		consumerName, // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	failOnError(err, "Failed to register a consumer")
	go consumeLoop(msgs, handlerFunc)
	go func() {
		KeepAliveLoop:
		for {
			select {
			case e := <-ch.NotifyClose(make(chan *amqp.Error)):
				log.Errorf("[RabbitMQ] Channel：%v:%v:%v:%v \n遇到问题已关闭，错误原因" + e.Error(), queueName,
					bindingKey,
					exchangeName,
					consumerName)
				for i := 0; i < 5; i++ {
					log.Infof("[RabbitMQ] 将在 8 s 后尝试重新注册 Channel: %v:%v:%v:%v", queueName,
						bindingKey,
						exchangeName,
						consumerName)
					time.Sleep(8 * time.Second)
					log.Debugf("[RabbitMQ] 开始注册 Channel：%v:%v:%v:%v", queueName,
						bindingKey,
						exchangeName,
						consumerName)
					err := func() (err error) {
						defer func() {
							if erron := recover(); erron != nil {
								switch erron := erron.(type) {
								case string:
									err = errors.New(erron)
								case error:
									err = erron
								case runtime.Error:
									err = erron
								default:
									panic(erron)
								}
							}
						}()
						err = m.Subscribe(
							exchangeName,
							exchangeType,
							queueName,
							consumerName,
							bindingKey,
							handlerFunc,
						)
						return
					}()
					if err == nil {
						break KeepAliveLoop
					} else {
						log.Error("[RabbitMQ] 重注册失败，信息：" + err.Error())
						if i == 4 {
							log.Fatal("[RabbitMQ] 无法恢复 Channel，进程退出。")
						}
					}
				}
			}
		}
	}()

	return nil
}

// SubscribeToQueue registers a handler function for the named queue.
func (m *AmqpClient) SubscribeToQueue(queueName string, consumerName string, handlerFunc func(amqp.Delivery) error) error {
	ch, err := m.conn.Channel()
	failOnError(err, "Failed to open a channel")

	log.Printf("Declaring Queue (%s)", queueName)
	queue, err := ch.QueueDeclare(
		queueName, // name of the queue
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)
	failOnError(err, "Failed to register an Queue")

	msgs, err := ch.Consume(
		queue.Name,   // queue
		consumerName, // consumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	failOnError(err, "Failed to register a consumer")

	go consumeLoop(msgs, handlerFunc)
	return nil
}

// 如果允许的话，关闭 AMQP-broker 的链接
func (m *AmqpClient) Close() {
	if m.conn != nil {
		log.Infoln("尝试关闭 AMQP broker 链接")
		if m.conn.Close() != nil {
			log.Warn("无法关闭链接")
		}
	}
}

func consumeLoop(deliveries <-chan amqp.Delivery, handlerFunc func(d amqp.Delivery) error) {
	for d := range deliveries {
		// Invoke the handlerFunc func we passed as parameter.
		err := handlerFunc(d)
		if err != nil {
			log.Errorf("接收器传递了一个错误： %s \n", err)
			err = d.Ack(false)
			if err != nil { // 如果出错了的话，标记信息为未处理
				log.Fatalf("ack 失败 %s \n", err)
			}
		} else {
			err = d.Ack(true)
			if err != nil {
				log.Fatalf("ack 失败 %s \n", err)
			}
		}
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Error("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}
