package redis

import (
	"context"
	"encoding/json"
	"fmt"

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

	Standalone bool

	Client *redis.Client
	PubSub *redis.PubSub

	subscription chan []string
	events       *golly.EventChain
}

// Leave this in place we want to make sure we are initalized before the initalizers or
// non befores are called for now.
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
	return nil
}

func Publish(ctx golly.Context, channel string, payload interface{}) {
	p, _ := json.Marshal(payload)

	server.Client.Publish(ctx.Context(), channel, p)
}

func Run() {
	if err := golly.Boot(func(a golly.Application) error { return run(a) }); err != nil {
		panic(err)
	}
}

func run(a golly.Application) error {
	quit := make(chan struct{})

	a.Logger.Info("Booting redis pubsub listener")

	golly.Events().Add(golly.EventAppShutdown, func(golly.Context, golly.Event) error {
		close(quit)
		return nil
	})

	return server.Receive(a, quit)
}

func (s Redis) Receive(a golly.Application, quit <-chan struct{}) error {
	ctx, cancel := context.WithCancel(a.GoContext())

	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			a.Logger.Errorln("panic in redis receive: ", r)
		}
	}()

	var quitting = false

	pubsub := s.Client.PSubscribe(ctx, "*")

	for !quitting {
		select {
		case message := <-pubsub.Channel():
			event := Event{Channel: message.Channel}
			if err := json.Unmarshal([]byte(message.Payload), &event.Payload); err == nil {
				server.events.AsyncDispatch(a.NewContext(ctx), message.Channel, event)
			} else {
				a.Logger.Errorf("unable to unmarshal event: %#v\n", err)
			}
		case <-quit:
			cancel()
			quitting = true
		}
	}

	return nil
}
