# Golly Web Framework

This is a work in progress check back soon :)

## Getting Started

We are still fairly new and undergoing alot of changes, if you want to check out Golly in the absence of any cli-tools
we recommend you grab our skeleton app [Skeleton Application](https://github.com/slimloans/golly-skeleton). 

## Configuration

Golly leverages Viper configuration tool to load configuration from various sources. 

JSON configuration can be located in any files:
- `$PWD/<yourAppName>.json`
- `$HOME/<yourAppName>.json`

By default Viper in golly is also configured to load environments from ENV

- `my.json.path` => `MY_JSON_PATH`

## Golly Context vs Golly WebContext
Golly providers clean accesss to some shared objects through the context function. This context then gets wrapped into the webcontext for each requst. In order to keep responsibilities shared we recommend only your endpoints have access to the WebContext and the Context provided within gets passed down to additional applications.

## Plugins

### ORM
Golly comes prepackaged with the gorm ORM for information about gorm visit there documentation page
[GORM](https://gorm.io). Some or all of the default configuration can be overriden. On top of the default configuration Golly accepts a connection URL wiht the following ENV `DATABASE_URL` 

#### Usage
To use the ORM plugin add orm.Initializer to golly's initializer by default located in `app/initializers/intializers.go`. Once ORM is initialized you can get a connection with `orm.DB(ctx)` the context provided to ORM.

#### Default Configuration

	v.SetDefault(appName, map[string]interface{}{
		"db": map[string]interface{}{
			"host":     "127.0.0.1",
			"port":     "5432",
			"username": "app",
			"password": "password",
			"name":     appName,
			"driver":   "postgres",
		},
	})

### Redis

### Passport

### GQL

## Examples
Below is a list of examples:
- [Skeleton Application](https://github.com/slimloans/golly-skeleton)


