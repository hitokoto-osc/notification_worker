package rabbitmq

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type RabbitMQ struct {
	// The RabbitMQ connection between client and the server
	conn *amqp.Connection

	// config stores the current configuration based on the given profile
	config *Config

	// logger interface
	log Logger
}

// NewWrapper Create RabbitMQ wrapper
func NewWrapper (config *Config, logger Logger) *RabbitMQ {
	return &RabbitMQ{
		config: config,
		log:    logger,
	}
}

// Conn returns RMQ connection
func (r *RabbitMQ) Conn() *amqp.Connection {
	return r.conn
}

var channelShouldUpdateConn = make(chan int)

// Dial RabbitMQ Connection
func (r *RabbitMQ) Dial () (err error) {
	if r.config == nil {
		return errors.WithStack(errors.New("[rabbitMQ] config is missing"))
	}
	if r.conn, err = amqp.Dial(r.config.URI()); err != nil {
		return err
	}
	r.handleError()
	return err // nil
}

// NotifyClose registers a listener for close events either initiated by an error
// accompaning a connection.close method or by a normal shutdown.
// On normal shutdowns, the chan will be closed.
// To reconnect after a transport or protocol error, we should register a listener here and
// re-connect to server
func (r *RabbitMQ) handleError () {
	go func() {
	KeepAliveLoop:
		for {
			select {
			case err := <-r.conn.NotifyClose(make(chan *amqp.Error)):
				r.log.Error("[RabbitMQ] AMQP 连接丢失，错误信息" + err.Error())
				for i := 0; i < 5; i++ {
					r.log.Info("[RabbitMQ] AMQP 连接将在 5 秒后尝试重连接...")
					time.Sleep(5 * time.Second)
					e := func() (err error) {
						err = r.Dial()
						return
					}()
					if e != nil {
						r.log.Error("[RabbitMQ] AMQP 重连接失败，错误信息：" + e.Error())
					} else {
						r.log.Info("[RabbitMQ] AMQP 连接已成功重建")
						channelShouldUpdateConn <- 1 // notifyUpdateLoop
						break KeepAliveLoop
					}
					if i == 4 {
						r.log.Fatal("[RabbitMQ] AMQP 重连接次数过多，程序退出。")
					}
				}
			}
		}
	}()
}

// Shutdown close connection to
func (r *RabbitMQ) Shutdown() error {
	return shutdown(r.conn)
}

// RegisterSignalHandler watchs for interrupt signals
// and gracefully closes connection
func (r *RabbitMQ) RegisterSignalHandler() {
	registerSignalHandler(r)
}

// Session is holding the current Exchange, Queue,
// Binding Consuming and Publishing settings for enclosed
// rabbitmq connection
type Session struct {
	// Exchange declaration settings
	Exchange Exchange
	// Queue declaration settings
	Queue Queue
	// Binding options for current exchange to queue binding
	BindingOptions BindingOptions
	// Consumer options for a queue or exchange
	ConsumerOptions *ConsumerOptions
	// Publishing options for a queue or exchange
	PublishingOptions *PublishingOptions
}

// Closer interface is for handling reconnection logic in a sane way
// Every reconnection supported struct should implement those methods
// in order to work properly
type Closer interface {
	RegisterSignalHandler()
	Shutdown() error
}

// shutdown is a general closer function for handling close gracefully
// Mostly here for both consumers and producers
// After a reconnection scenerio we are gonna call shutdown before connection
func shutdown(conn *amqp.Connection) error {
	if err := conn.Close(); err != nil {
		if amqpError, isAmqpError := err.(*amqp.Error); isAmqpError && amqpError.Code != 504 {
			return fmt.Errorf("AMQP connection close error: %s", err)
		}
	}
	return nil
}

// shutdownChannel is a general closer function for channels
func shutdownChannel(channel *amqp.Channel, tag string) error {
	// This waits for a server acknowledgment which means the sockets will have
	// flushed all outbound publishings prior to returning.  It's important to
	// block on Close to not lose any publishings.
	if err := channel.Cancel(tag, true); err != nil {
		if amqpError, isAmqpError := err.(*amqp.Error); isAmqpError && amqpError.Code != 504 {
			return fmt.Errorf("AMQP connection close error: %s", err)
		}
	}

	if err := channel.Close(); err != nil {
		return err
	}

	return nil
}


// registerSignalHandler helper function for stopping consumer or producer from
// operating further
// Watchs for SIGINT, SIGTERM, SIGQUIT, SIGSTOP and closes connection
func registerSignalHandler(c Closer) {
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals)
		for {
			s := <-signals
			switch s {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				err := c.Shutdown()
				if err != nil {
					panic(err)
				}
				os.Exit(0)
			}
		}
	}()
}
