/*
Package golly is an ergonomic, service framework designed for production engineering.

# Overview

Golly is not just a web framework. It provides the architecture to build scalable systems where your API,
Background Workers, and Consumer workloads live in a single, cohesive codebase but scale independently.
It prioritizes "Monolithic Logic, Independent Scale".

# Key Features

  - Service Framework: Define multiple Services (Web, Worker, Consumer) in one binary.
  - Zero-Allocation: Obsessively optimized logger, router, and context for hot paths.
  - Supercharged Context: Carries structured logging, identity, and safe detachment logic.
  - Plugin Ecosystem: Modular architecture for ORM, Kafka, Redis, etc.
  - Production Ready: Graceful shutdown, lifecycle hooks, and built-in CLI.

# Usage

A minimal example of a Golly application entrypoint:

	package main

	import (
		"github.com/golly-go/golly"
		"github.com/golly-go/golly/middleware"
	)

	func main() {
		golly.Run(golly.Options{
			Name: "my-app",
			Initializer: func(app *golly.Application) error {
				app.Routes().
					Use(middleware.RequestLogger).
					Get("/", func(c *golly.WebContext) {
						c.Text(200, "Hello Golly!")
					})
				return nil
			},
		})
	}

# Components

  - Application: The root container for configuration, plugins, and lifecycle.
  - Context: The request-scoped context carrying metadata and logger.
  - Logger: A high-performance, structured logger (zero-alloc).
  - WebContext: A pool-optimized wrapper for HTTP requests.

For more details, see the README or visit https://github.com/golly-go/golly.
*/
package golly
