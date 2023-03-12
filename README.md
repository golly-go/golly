# Golly Web Framework

This is a work in progress check back soon :)

## Getting Started

We are still fairly new and undergoing alot of changes, if you want to check out Golly in the absence of any cli-tools
we recommend you grab our skeleton app [Skeleton Application](https://github.com/golly-go/golly-skeleton). 

## Configuration

Golly leverages [Viper](https://github.com/spf13/viper) configuration tool to load configuration from various sources. 

JSON configuration can be located in any files:
- `$PWD/<yourAppName>.json`
- `$HOME/<yourAppName>.json`

By default Viper in golly is also configured to load environments from ENV

- `MY_JSON_PATH` => `my.json.path` in the code

## Golly Context vs Golly WebContext
Golly providers clean accesss to some shared objects through the context function. This context then gets wrapped into the webcontext for each requst. In order to keep responsibilities shared we recommend only your endpoints have access to the WebContext and the Context provided within gets passed down to additional applications.


