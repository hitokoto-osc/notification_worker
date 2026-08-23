package rabbitmq

import "sync"

// producerCache 缓存 routing key 到 producer UUID 的映射。
//
// 每条 delivery 都会派生独立的 goroutine，因此这里必须是并发安全的。
// 锁在 create 期间保持不释放，以此获得 single-flight 语义：并发的缓存
// 未命中不会各自建立一条 AMQP channel 而造成泄漏。
type producerCache struct {
	mu      sync.Mutex
	entries map[string]string
}

func newProducerCache() *producerCache {
	return &producerCache{entries: make(map[string]string)}
}

// getOrCreate 返回 routing key 对应的 producer，必要时通过 create 建立。
// lookup 用于校验缓存项是否仍然有效，返回 false 时该缓存项会被丢弃。
func (c *producerCache) getOrCreate(
	routingKey string,
	lookup func(uuid string) (*Producer, bool),
	create func() (*Producer, error),
) (*Producer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if uuid, ok := c.entries[routingKey]; ok {
		if producer, alive := lookup(uuid); alive {
			return producer, nil
		}
		delete(c.entries, routingKey)
	}

	producer, err := create()
	if err != nil {
		return nil, err
	}
	c.entries[routingKey] = producer.UUID
	return producer, nil
}

var producersCache = newProducerCache()
