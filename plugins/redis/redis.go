package redis

import (
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

	EnableSubscription bool

	Client *redis.Client
	PubSub *redis.PubSub

	subscription chan []string
	events       *golly.EventChain
}

func init() {
	golly.RegisterRunner("pubsub", golly.Runner{Handler: runner})

	golly.
		Events().
		Add(golly.EventAppBeforeInitalize, BeforeInitialize)
}

func BeforeInitialize(gctx golly.Context, evt golly.Event) error {
	server = Redis{
		EnableSubscription: gctx.RunMode() == "pubsub",
		subscription:       make(chan []string),
		events:             &golly.EventChain{},
		Client: redis.NewClient(&redis.Options{
			Addr:     gctx.Config().GetString("redis.address"),
			Password: gctx.Config().GetString("redis.password"),
			DB:       0,
		}),
	}

	server.PubSub = server.Client.Subscribe(server.GoContext())

	return nil
}

func Server() Redis {
	return server
}

func Subscribe(handler golly.EventHandlerFunc, channels ...string) error {
	if server.EnableSubscription {
		for _, channel := range channels {
			server.events.Add(channel, handler)
		}
		server.subscription <- channels
	}
	return nil
}

func Publish(ctx golly.Context, channel string, payload interface{}) {
	p, _ := json.Marshal(payload)

	server.Client.Publish(ctx.Context(), channel, p)
}

func runner(a golly.Application) error {
	quit := make(chan struct{})

	golly.Events().Add(golly.EventAppShutdown, func(golly.Context, golly.Event) error {
		close(quit)
		return nil
	})

	return server.Receive(a, quit)
}

func (s Redis) Receive(a golly.Application, quit <-chan struct{}) error {
	var quitting = false

	ch := s.PubSub.Channel()

	for !quitting {
		select {
		case message := <-ch:
			fmt.Printf("Message: %#v\n", message)
		case channels := <-s.subscription:
			s.PubSub.PSubscribe(a.GoContext(), channels...)
		case <-quit:
			quitting = true
		}
	}

	return nil
}
