package rabbitmq

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChannelState struct {
	closed atomic.Bool
}

func (s *fakeChannelState) IsClosed() bool { return s.closed.Load() }

func newFakeProducer(uuid string) (*Producer, *fakeChannelState) {
	state := &fakeChannelState{}
	return &Producer{UUID: uuid, channelState: state}, state
}

// 并发缓存未命中必须只创建一个 producer，否则会泄漏 AMQP channel。
func TestProducerCacheGetOrCreateIsSingleFlight(t *testing.T) {
	cache := newProducerCache()
	producer, _ := newFakeProducer("producer-1")

	var created atomic.Int64
	const callers = 64

	start := make(chan struct{})
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got, err := cache.getOrCreate(
				"notification.queue",
				func(uuid string) (*Producer, bool) { return producer, uuid == producer.UUID },
				func() (*Producer, error) {
					created.Add(1)
					return producer, nil
				},
			)
			assert.NoError(t, err)
			assert.Same(t, producer, got)
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int64(1), created.Load())
}

// 缓存项失效时应被丢弃并重建，而不是反复返回已死的 producer。
func TestProducerCacheGetOrCreateReplacesStaleEntry(t *testing.T) {
	cache := newProducerCache()
	stale, _ := newFakeProducer("stale")
	fresh, _ := newFakeProducer("fresh")

	got, err := cache.getOrCreate("key",
		func(string) (*Producer, bool) { return nil, false },
		func() (*Producer, error) { return stale, nil },
	)
	require.NoError(t, err)
	require.Same(t, stale, got)

	got, err = cache.getOrCreate("key",
		func(string) (*Producer, bool) { return nil, false },
		func() (*Producer, error) { return fresh, nil },
	)
	require.NoError(t, err)
	assert.Same(t, fresh, got)
}

// Get 命中已关闭的 channel 时需要回收该条目——旧实现在持有读锁时调用
// Remove()（需要写锁），因 RWMutex 不可升级而自死锁。
func TestProducerListGetEvictsClosedProducer(t *testing.T) {
	producer, state := newFakeProducer("producer-1")

	var list ProducerList
	list.Add(ProducerUnit{UUID: producer.UUID, Producer: producer})

	got, ok := list.Get(producer.UUID)
	require.True(t, ok)
	require.Same(t, producer, got)

	state.closed.Store(true)
	got, ok = list.Get(producer.UUID)
	assert.False(t, ok)
	assert.Nil(t, got)
	assert.Empty(t, list.list)
}

func TestProducerListConcurrentGetAddRemove(t *testing.T) {
	producer, _ := newFakeProducer("producer-1")

	var list ProducerList
	list.Add(ProducerUnit{UUID: producer.UUID, Producer: producer})

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 100 {
				list.Get(producer.UUID)
			}
		}()
		go func() {
			defer wg.Done()
			for range 100 {
				list.Remove(producer.UUID)
				list.Add(ProducerUnit{UUID: producer.UUID, Producer: producer})
			}
		}()
	}
	wg.Wait()
}

// Remove 旧实现在 range 中用 append 切片，会跳过被移除元素之后的元素。
func TestRemoveUnitByUUIDRemovesEveryMatch(t *testing.T) {
	list := []ProducerUnit{
		{UUID: "a"}, {UUID: "dup"}, {UUID: "dup"}, {UUID: "b"}, {UUID: "dup"},
	}
	got := removeUnitByUUID(list, "dup", func(u ProducerUnit) string { return u.UUID })

	require.Len(t, got, 2)
	assert.Equal(t, "a", got[0].UUID)
	assert.Equal(t, "b", got[1].UUID)
}
