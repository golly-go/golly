package redis

import (
	"context"
	"encoding/json"
	"fmt"

	redis "github.com/go-redis/redis/v8"
	"github.com/slimloans/golly"
)

var (
	server Redis
)

type Event struct {
	Channel       string
	Payload       map[string]interface{}
	PayloadString string
}

type Redis struct {
	*redis.Client

	PubSub *redis.PubSub

	events *golly.EventChain
}

func Initializer(address, password string) golly.GollyAppFunc {
	return func(a golly.Application) error {
		a.Logger.Infof("Redis connection initalized to %s", address)

		server := Redis{
			events: &golly.EventChain{},
			Client: redis.NewClient(&redis.Options{
				Addr:     address,
				Password: password,
				DB:       0,
			}),
		}

		if server.Client == nil {
			return fmt.Errorf("unable to initalize redis")
		}

		return nil
	}
}

func Client() Redis {
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

// This is the next thing we need to manage this doesnt quite work 100% and with golly's
// concept of run modes to allow seperation of app and resources, we need a better way of handling
// this
func (s Redis) Receive(a golly.Application, quit <-chan struct{}) error {
	var quitting = false

	ctx, cancel := context.WithCancel(a.GoContext())

	defer func() {
		if r := recover(); r != nil {
			a.Logger.Errorln("panic in redis receive: ", r)
		}
		cancel()
	}()

	pubsub := s.Client.PSubscribe(ctx, "*")

	for !quitting {
		select {
		case message := <-pubsub.Channel():
			event := Event{Channel: message.Channel, PayloadString: message.Payload}

			if err := json.Unmarshal([]byte(message.Payload), &event.Payload); err == nil {
				server.events.AsyncDispatch(a.NewContext(ctx), message.Channel, event)
			}
		case <-quit:
			quitting = true
		}
	}
	return nil
}
