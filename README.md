# Golly: Lightweight Web Framework for Go

Welcome to **Golly**, a lightweight, ergonomic, and extensible web framework for building modern web applications in Go. Golly provides powerful tools and patterns to simplify the development of APIs, middleware, and web services, while maintaining performance and minimalism.

## Key Features

### 🔌 Plugin System
Golly supports a plugin-based architecture for extensibility. Easily integrate additional functionality such as ORM, caching, or messaging with custom plugins. Check out the [Golly Plugins](https://github.com/golly-go/plugins) for official integrations.

### 🧰 Middleware Support
Integrate middleware for logging, recovery, CORS, and more. Middleware in Golly is first-class and designed to be composable and reusable.

### 📜 Declarative Routing
Define routes declaratively with support for middleware chaining, handlers, and dynamic paths.

### 🚀 High Performance
Golly prioritizes performance by minimizing allocations and providing optimized utilities, ensuring your applications are fast and scalable.

### 🛠 Utility Functions
Includes ergonomic utilities for common tasks like rendering responses, marshaling data, and error handling.

## Installation

Add Golly to your project:

```bash
# Add Golly
go get github.com/golly-go/golly

# Add Plugins (optional)
go get github.com/golly-go/plugins
```

## Getting Started

Here’s a simple example to get you started:

```go
package main

import (
	"net/http"

	"github.com/golly-go/golly"
	"github.com/golly-go/golly/middleware"
	"github.com/golly-go/plugins/orm"
)

func main() {
	golly.Run(golly.Options{
		Plugins: []golly.Plugin{
			orm.NewOrmPlugin(orm.SQLiteConfig{InMemory: true}),
		},
		Initializers: []golly.AppFunc{initializer},
	})
}

func initializer(app *golly.Application) error {
	app.Routes().
		Use(middleware.RequestLogger).
		Use(middleware.Recoverer).
		Use(middleware.Cors(middleware.CorsOptions{
			AllowedOrigins: []string{"http://localhost:9000"},
		})).
		Get("/hello", func(wctx *golly.WebContext) {
			golly.Render(wctx, golly.FormatTypeText, "Hello, World!")
		})

	return nil
}
```

### Example Breakdown
- **Plugins**: Adds an in-memory SQLite database using the ORM plugin.
- **Middleware**: Logs requests, recovers from panics, and applies CORS policies.
- **Routes**: Defines a single `/hello` endpoint that returns `Hello, World!`.

## Advanced Features

### Plugins
Extend Golly with plugins to integrate additional functionality:

```go
ormPlugin := orm.NewOrmPlugin(orm.PostgresConfig{
	Host:     "localhost",
	User:     "postgres",
	Password: "password",
	Database: "mydb",
	Port:     5432,
})

golly.Run(golly.Options{
	Plugins: []golly.Plugin{ormPlugin},
})
```

### Middleware
Middleware is chainable and composable:

```go
app.Routes().
	Use(middleware.RequestLogger).
	Use(middleware.Recoverer).
	Use(myCustomMiddleware).
	Get("/api", apiHandler)
```

### Custom Handlers
Handlers are simple to define:

```go
func apiHandler(wctx *golly.WebContext) {
	golly.RenderJSON(wctx, map[string]string{
		"message": "This is an API response",
	})
}
```

### Render Helpers
Golly provides helper functions for rendering responses, these helpers add minimal time to the response and are in hopes to provide better egonomics when using 
golly for your project, but they are not required so dont feel forced to use them

- **`Render`**: Automatically marshals data based on format.
- **`RenderJSON`**: Marshals data into JSON.
- **`RenderXML`**: Marshals data into XML.
- **`RenderText`**: Renders plain text.
- **`RenderData`**: Renders Bytes (sending a file etc)

```go
golly.RenderJSON(wctx, map[string]string{"key": "value"})
```

## Middleware

Golly comes with built-in middleware to handle common needs like logging, CORS, and recovery.

### CORS Middleware

The CORS middleware enables cross-origin resource sharing for your routes. It can be configured using the `CorsOptions` struct:

#### Options:

- `AllowAllHeaders` (bool): If `true`, allows all headers.
- `AllowAllOrigins` (bool): If `true`, allows all origins.
- `AllowedHeaders` ([]string): Specifies which headers are allowed.
- `AllowedMethods` ([]string): Specifies which HTTP methods are allowed.
- `AllowedOrigins` ([]string): Specifies which origins are allowed.
- `ExposeHeaders` ([]string): Specifies which headers are exposed to the browser.
- `AllowCredentials` (bool): If `true`, includes credentials (e.g., cookies) in requests.

#### Example:

```go
app.Routes().
	Use(middleware.Cors(middleware.CorsOptions{
		AllowedOrigins: []string{"http://example.com", "http://localhost:9000"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		ExposeHeaders:  []string{"X-Custom-Header"},
		AllowCredentials: true,
	}))
```

### Request Logger

Logs details about each incoming HTTP request, including method, URL, status, and duration.

### Recoverer

Catches panics in your application and returns a `500 Internal Server Error` response while logging the error.

---

## Contributing

Contributions are welcome! Feel free to open issues, submit pull requests, or suggest features.

1. Fork the repo.
2. Create a feature branch.
3. Submit a pull request.

## Community and Support

- [Plugins Repository](https://github.com/golly-go/plugins)
- [Issue Tracker](https://github.com/golly-go/golly/issues)

---

**Happy coding with Golly!**

