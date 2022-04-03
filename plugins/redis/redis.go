package redis

import (
	"context"
	"encoding/json"
	"fmt"

	redis "github.com/go-redis/redis/v8"
	"github.com/slimloans/golly"
	"github.com/slimloans/golly/errors"
)

var (
	server Redis = newRedis()
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

func newRedis() Redis {
	return Redis{
		events: &golly.EventChain{},
	}
}

func Preboot() error {
	golly.RegisterInitializerEx(true, initializer)
	return nil
}

func config(a golly.Application) (string, string, int) {
	a.Config.SetDefault("redis", map[string]string{
		"password": "",
		"address":  "127.0.0.1:6379",
		"db":       "0",
	})

	return a.Config.GetString("redis.address"),
		a.Config.GetString("redis.password"),
		a.Config.GetInt("redis.db")
}

func initializer(a golly.Application) error {
	address, password, db := config(a)

	a.Logger.Infof("Redis connection initalized to %s", address)

	server.Client = redis.NewClient(
		&redis.Options{
			Addr:     address,
			Password: password,
			DB:       db,
		})

	return nil
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
	// Guard against missconfigured
	if server.Client != nil {
		p, _ := json.Marshal(payload)
		server.Client.Publish(ctx.Context(), channel, p)
	}
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

	if s.Client == nil {
		a.Logger.Error("redis client is nil check to see if the initializer has been ran.")
		return errors.Wrap(errors.ErrorMissConfigured, fmt.Errorf("redis is not configured correct"))
	}

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
