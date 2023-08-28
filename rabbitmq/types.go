package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

// Config is the rabbitmq connect profile
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Vhost    string
}

// URI convert config to amqp uri
func (c *Config) URI() string {
	return amqp.URI{
		Scheme:   "amqp",
		Host:     c.Host,
		Port:     c.Port,
		Username: c.Username,
		Password: c.Password,
		Vhost:    c.Vhost,
	}.String()
}

// Exchange defines rabbitMQ Exchange policy
type Exchange struct {
	// Exchange name
	Name string
	// Exchange type
	Type string
	// Durable exchanges will survive server restarts
	Durable bool
	// Will remain declared when there are no remaining bindings.
	AutoDelete bool
	// Exchanges declared as `internal` do not accept accept publishings.Internal
	// exchanges are useful for when you wish to implement inter-exchange topologies
	// that should not be exposed to users of the broker.
	Internal bool
	// When noWait is true, declare without waiting for a confirmation from the server.
	NoWait bool
	// amqp.Table of arguments that are specific to the server's implementation of
	// the exchange can be sent for exchange types that require extra parameters.
	Args amqp.Table
}

// Queue defines rabbitMQ Queue policy
type Queue struct {
	// The queue name may be empty, in which the server will generate a unique name
	// which will be returned in the Name field of Queue struct.
	Name string
	// Check Exchange comments for durable
	Durable bool
	// Check Exchange comments for autodelete
	AutoDelete bool
	// Exclusive queues are only accessible by the connection that declares them and
	// will be deleted when the connection closes.  Channels on other connections
	// will receive an error when attempting declare, bind, consume, purge or delete a
	// queue with the same name.
	Exclusive bool
	// When noWait is true, the queue will assume to be declared on the server.  A
	// channel exception will arrive if the conditions are met for existing queues
	// or attempting to modify an existing queue from a different connection.
	NoWait bool
	// Check Exchange comments for Args
	Args amqp.Table
}

// BindingOptions contains options of binding policy
type BindingOptions struct {
	// Publishings messages to given Queue with matching -RoutingKey-
	// Every Queue has a default binding to Default Exchange with their Qeueu name
	// So you can send messages to a queue over default exchange
	RoutingKey string
	// Do not wait for a consumer
	NoWait bool
	// App specific data
	Args amqp.Table
}
