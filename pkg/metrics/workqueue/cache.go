package workqueue

import "sync"

type cache struct {
	store map[string]interface{}
	lock  sync.Mutex
}

func (c *cache) GetOrCreate(key string, createFunc func() interface{}) interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.store == nil {
		c.store = make(map[string]interface{})
	}

	if entry, ok := c.store[key]; ok {
		return entry
	}

	obj := createFunc()
	c.store[key] = obj
	return obj
}
