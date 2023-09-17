package rabbitmq

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Producer is RabbitMQ Producer wrapper
// TODO: 修复 done 的信道的使用
type Producer struct {
	// Producer UUID
	UUID string
	// instance is a pointer to Instance that Producer belongs to
	instance *Instance
	// Base struct for Producer
	RabbitMQ *RabbitMQ
	// The communication channel over connection
	channel *amqp.Channel
	// A notifiyng channel for publishings
	done chan error
	// Current producer connection settings
	session Session
}

type PublishingOptions struct {
	// The key that when publishing a message to a exchange/queue will be only delivered to
	// given routing key listeners
	RoutingKey string
	// Publishing tag
	Tag string
	// Queue should be on the server/broker
	Mandatory bool
	// Consumer should be bound to server
	Immediate bool
}

// NewProducer is a constructor function for producer creation Accepts Exchange,
// Queue, PublishingOptions. On the other hand we are not declaring our topology
// on both the publisher and consumer to be able to change the settings only in
// one place. We can declare those settings on both place to ensure they are
// same. But this package will not support it.
func (r *RabbitMQ) NewProducer(instance *Instance, e Exchange, q Queue, po PublishingOptions) (*Producer, error) {

	if r.Conn() == nil {
		return nil, errors.WithStack(errors.New("[RabbitMQ.Producer] RabbitMQ Connection is missing"))
	}

	// getting a channel
	mutex.Lock()
	channel, err := r.conn.Channel()
	mutex.Unlock()
	if err != nil {
		return nil, errors.Wrap(err, "[RabbitMQ.Producer] Channel creation error")
	}
	uuidInstance, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "[RabbitMQ.Producer] UUID generation error")
	}
	producer := &Producer{
		UUID:     uuidInstance.String(),
		RabbitMQ: r,
		instance: instance,
		channel:  channel,
		session: Session{
			Exchange:          e,
			Queue:             q,
			PublishingOptions: &po,
		},
	}
	producer.HandleError()
	return producer, nil
}

// HandleError register the recover loop against channel closed
func (p *Producer) HandleError() {
	go func() {
	loop:
		for {
			select {
			case e := <-p.channel.NotifyClose(make(chan *amqp.Error)):
				p.RabbitMQ.log.Errorf("[RabbitMQ] Channel：%+v \n遇到问题已关闭，错误原因：%v", p.session, e.Error())
				break loop
			}
		}
	}()
}

func (p *Producer) GetRoutingKey() string {
	routingKey := p.session.PublishingOptions.RoutingKey
	// if exchange name is empty, this means we are gonna publish
	// this mesage to a queue, every queue has a binding to default exchange
	if p.session.Exchange.Name == "" {
		routingKey = p.session.Queue.Name
	}
	return routingKey
}

// Publish sends a Publishing from the client to an exchange on the server.
func (p *Producer) Publish(ctx context.Context, publishing amqp.Publishing) error {
	e := p.session.Exchange
	po := p.session.PublishingOptions
	routingKey := p.GetRoutingKey()
	err := p.channel.PublishWithContext(
		ctx,
		e.Name,       // publish to an exchange(it can be default exchange)
		routingKey,   // routing to 0 or more queues
		po.Mandatory, // mandatory, if no queue than err
		po.Immediate, // immediate, if no consumer than err
		publishing,
		// amqp.Publishing {
		//        // Application or exchange specific fields,
		//        // the headers exchange will inspect this field.
		//        Headers Table

		//        // Properties
		//        ContentType     string    // MIME content type
		//        ContentEncoding string    // MIME content encoding
		//        DeliveryMode    uint8     // Transient (0 or 1) or Persistent (2)
		//        Priority        uint8     // 0 to 9
		//        CorrelationId   string    // correlation identifier
		//        ReplyTo         string    // address to to reply to (ex: RPC)
		//        Expiration      string    // message expiration spec
		//        MessageId       string    // message identifier
		//        Timestamp       time.Time // message timestamp
		//        Type            string    // message type name
		//        UserId          string    // creating user id - ex: "guest"
		//        AppId           string    // creating application id

		//        // The application specific payload of the message
		//        Body []byte
		// }
	)

	return err
}

// NotifyReturn captures a message when a Publishing is unable to be
// delivered either due to the `mandatory` flag set
// and no route found, or `immediate` flag set and no free consumer.
func (p *Producer) NotifyReturn(notifier func(message amqp.Return)) {
	go func() {
		for res := range p.channel.NotifyReturn(make(chan amqp.Return)) {
			notifier(res)
		}
	}()

}

// Shutdown gracefully closes all connections
func (p *Producer) Shutdown() error {
	po := p.session.PublishingOptions
	if po == nil {
		return errors.WithStack(errors.New("[RabbitMQ.Producer] PublishingOptions is missing"))
	}
	if err := shutdownChannel(
		p.channel,
		po.Tag,
	); err != nil {
		return err
	}

	// Since publishing is asynchronous this can happen
	// instantly without waiting for a done message.
	defer p.RabbitMQ.log.Info("Producer shutdown OK")
	defer p.RabbitMQ.log.Sync()
	return nil
}
