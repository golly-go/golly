package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/slimloans/golly"
)

type Event struct {
	Channel string
	Payload map[string]interface{}
}

var (
	server Redis
)

type Redis struct {
	*golly.Application

	Client *redis.Client
	PubSub *redis.PubSub

	subscription chan []string
	events       *golly.EventChain
}

func init() {
	golly.
		Events().
		Add(golly.EventAppBeforeInitalize, BeforeInitialize)
}

func BeforeInitialize(gctx golly.Context, evt golly.Event) error {
	gctx.Config().SetDefault("redis", map[string]string{
		"password": "",
		"address":  "127.0.0.1:6379",
	})

	addr := gctx.Config().GetString("redis.address")
	gctx.Logger().Infof("Redis connection initalized to %s (Run Mode: %s)", addr, gctx.RunMode())

	server = Redis{
		subscription: make(chan []string, 10),
		events:       &golly.EventChain{},
		Client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: gctx.Config().GetString("redis.password"),
			DB:       0,
		}),
	}

	if server.Client == nil {
		return fmt.Errorf("unable to initalize redis")
	}

	return nil
}

func Server() Redis {
	return server
}

func Subscribe(handler golly.EventHandlerFunc, channels ...string) error {
	for _, channel := range channels {
		server.events.Add(channel, handler)
	}
	server.subscription <- channels
	return nil
}

func Publish(ctx golly.Context, channel string, payload interface{}) {
	p, _ := json.Marshal(payload)

	server.Client.Publish(ctx.Context(), channel, p)
}

func runner(a golly.Application) error {
	quit := make(chan struct{})
	a.Logger.Info("Booting redis pubsub listener")

	golly.Events().Add(golly.EventAppShutdown, func(golly.Context, golly.Event) error {
		close(quit)
		return nil
	})

	return server.Receive(a, quit)
}

func (s Redis) Receive(a golly.Application, quit <-chan struct{}) error {
	ch := make(chan *redis.Message, 100)
	ctx, cancel := context.WithCancel(a.GoContext())

	defer func() {
		if r := recover(); r != nil {
			a.Logger.Errorln("panic in redis receive: ", r)
		}
	}()

	var quitting = false

	pubsub := s.Client.Subscribe(ctx)

	go func(ctx context.Context, ch chan *redis.Message) {
		for ctx.Err() == nil {
			if message, _ := pubsub.ReceiveMessage(ctx); message != nil {
				ch <- message
			}
			time.Sleep(1 * time.Millisecond)
		}
	}(ctx, ch)

	for !quitting {
		select {
		case message := <-ch:
			fmt.Printf("Message: %#v\n", message)
		case channels := <-s.subscription:
			fmt.Printf("%#v\n", channels)

			pubsub.Subscribe(a.GoContext(), channels...)
		case <-quit:
			cancel()
			quitting = true
		}
	}

	return nil
}
