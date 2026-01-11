# Golly

> **The Go Service Framework for High-Performance Monoliths**

[![Go Reference](https://pkg.go.dev/badge/github.com/golly-go/golly.svg)](https://pkg.go.dev/github.com/golly-go/golly)
[![Go Report Card](https://goreportcard.com/badge/github.com/golly-go/golly)](https://goreportcard.com/report/github.com/golly-go/golly)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> [!WARNING] > **v0.9.1 Breaking Changes**
> This release introduces significant architectural improvements that require migration:
>
> - **Logger**: Replaced `logrus` with a zero-allocation custom logger. Syntax has changed (see `doc.go`).
> - **Context**: `WebContext` no longer implements `context.Context` directly to prevent unintended interface promotion. Use `wctx.Context()` to access the underlying context.

**Golly** is an ergonomic, zero-allocation service framework designed for **production engineering**.

Most Go frameworks are just HTTP routers. Golly is different. It provides the **architecture** to build scalable systems where your API, Background Workers, and Consumer workloads live in a single, cohesive codebase but scale independently.

---

## The Philosophy: Monolithic Logic, Independent Scale

Stop tearing your application apart into microservices just to scale a queue consumer.

Golly allows you to define multiple **Services** (Web, Worker, Consumer) in one binary. In production, you deploy the same binary as different "Pods", each running only the workload it needs.

- **Pod A (Web)**: API Traffic, highly scalable, stateless.
- **Pod B (Worker)**: Background jobs, memory-intensive.
- **Pod C (Consumer)**: Kafka stream processor, throughput-optimized.

All sharing the same models, domain logic, and utilities. **Zero logic duplication. Infinite scale.**

---

## Obsessive Performance

We don't just say "fast"; we prove it. Golly is engineered for **High-Performance** in the hot paths.

| Component  | Operation          | Latency     | Allocations   |
| :--------- | :----------------- | :---------- | :------------ |
| **Logger** | `Log.Opt().Info()` | **~133 ns** | **0 allocs**  |
| **Router** | `GET /blog/:id`    | **~136 ns** | **1 alloc**\* |
| **CORS**   | Origin Validation  | **~12 ns**  | **0 allocs**  |
| **Render** | JSON Response      | **~157 ns** | **3 allocs**  |

_\*The single router allocation is the `Context` object, ensuring thread-safe context propagation across different execution modes._

---

## Real World Production Example

Golly applications are declarative and easy to reason about. Here is what a real-world production entrypoint looks like:

```go
package main

import (
	"time"

	"github.com/golly-go/golly"
	"github.com/golly-go/golly/middleware"
	"github.com/golly-go/plugins/orm"
	"github.com/golly-go/plugins/eventsource"
	"github.com/golly-go/plugins/kafka"
)

func main() {
	golly.Run(golly.Options{
		Name:    "hris-backend",
		Version: "1.0.0",

		// Define functional workloads that can run in this binary
		Services: []golly.Service{
			&golly.WebService{},      // HTTP API
			&worker.JobProcessor{},   // Background Jobs
			&kafka.ConsumerService{}, // Message Consumer
		},

		// Structured, ordered initialization chain
		Initializer: golly.AppFuncChain(
			lib.Initializer,
			domains.Initializer,
			infra.Initializer,
		),

		// Drop-in Plugins for enterprise capabilities
		Plugins: []golly.Plugin{
			// Database with automatic configuration
			orm.NewOrmPlugin(func(app *golly.Application) (orm.PostgresConfig, error) {
				return orm.PostgresConfig{
					Host:         app.Config().GetString("db.host"),
					MaxOpenConns: 100,
					YAMLConfig:   true, // Load from config/database.yml
				}, nil
			}),

			// Event Sourcing & Kafka
			kafka.NewPlugin(),
			eventsource.NewPlugin(
				eventsource.PluginWithStore(&gormstore.Store{}),
			),
		},
	})
}
```

---

## Key Features

### 1. The Supercharged Context

Golly's `Context` is the spine of your request. It's not just a bag of values; it's an intelligent carrier for:

- **Structured Logging**: `ctx.Logger()` inherits request IDs and tenant info automatically.
- **Identity**: Built-in methods for `ctx.Actor()` and authentication state.
- **Safe Detachment**: Use `ctx.Detach()` to spawn goroutines that keep trace metadata but survive request cancellation.

### 2. High-Performance Logger

Why use `zap` when you can have something built-in and cleaner?

```go
// Near-Zero Heap Allocations. Typesafe.
logger.Opt().
    Str("service", "payment").
    Int("status", 200).
    Dur("latency", duration).
    Info("Payment processed")
```

### 3. CLI Built-in

Every Golly app is also a CLI. Need to run a migration? Truncate a table?

```bash
# Run the web server
./app start

# Run a specific service
./app service start worker

# Run custom commands
./app db truncate
```

### 4. Plugin Ecosystem

Don't write boilerplate. Use the official plugins:

- `golly-go/plugins/orm`: GORM integration with connection pooling.
- `golly-go/plugins/kafka`: High-throughput consumers/producers.
- `golly-go/plugins/redis`: Cache and PubSub.
- `golly-go/plugins/eventsource`: CQRS and Event Sourcing patterns.

---

## Installation

```bash
go get github.com/golly-go/golly
```

---

## Contributing

We are building the framework we always wanted to use. If you care about performance, code aesthetics, and developer happiness, join us.

1.  Fork the repo.
2.  Create your feature branch.
3.  Submit a Pull Request.
