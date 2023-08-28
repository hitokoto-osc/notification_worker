package context

import "context"

type IContext interface {
	Get(key string) interface{}
	Set(key string, value interface{})
}

type Context struct {
	context.Context
	m map[string]interface{}
}

func NewFromContext(ctx context.Context) context.Context {
	return &Context{
		Context: ctx,
		m:       make(map[string]interface{}),
	}
}

func New() context.Context {
	return NewFromContext(context.Background())
}

func (c *Context) Get(key string) interface{} {
	return c.m[key]
}

func (c *Context) Set(key string, value interface{}) {
	c.m[key] = value
}
