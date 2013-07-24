# wayang

## What It Is

`wayang` is an app that very naïvely mocks REST APIs, primarily designed for testing your frontend without a fully-working backend. It can be used standalone for a single project, or hosted as a pseudo-SaaS when combined with MongoDB (and possibly other database systems in future).

You supply `wayang` with JSON configuration that dictates how the mocked endpoints should behave, and wayang does exactly what you ask it to do (barring bugs; report these in the issues tab please).

## What It Isn't

`wayang` isn't meant to be a BaaS (Backend-as-a-Service), so if you want your `POST`s, `PUT`s, `PATCH`es, and `DELETE`s to actually do something, you'll want to try one of the various BaaS providers out there.

# Configuration

`wayang` listens on `:8000` by default; this can be overridden with the -laddr flag.

`wayang` looks for `config.json` in your current directory by default, but this can be overridden with the `-conf` flag.

The configuration JSON file is required to contain two keys, `db` and `db_addr`.

Currently supported values for `db` are `mongodb` and `static`.

`db_addr` is the address of the database to connect to in any case except for `static`, where it is the path to the static configuration file.

Additionally, the configuration JSON may contain the `http_prefix` key. This essentially prefixes the specified value to all your API endpoints.

The primary use case for this is if you're hosting `wayang` at, say, `https://derp.herp/mocks/`. In order for `wayang`'s routing to work properly, you would have to set `http_prefix` to `/mocks`.

# Usage

## Static Mode

In static mode, mock-config JSON is read from the file pointed to by `db_addr` in the configuration JSON.

To re-load the mock-config without restarting the app, do a `GET` request to `https://hostname/http_prefix/__update__`. This call may take a while if your JSON is large.

## Pseudo-SaaS Mode

With any other database backend, `wayang` will create a new mock root at a URL with a db-specific prefix. In the case of MongoDB, the URL would be `https://hostname/http_prefix/<mongo_id_here>`

# Configuring Mocks

The mock-config JSON used in both static mode and POSTed to the root endpoint in psueudo-SaaS mode has the following format:

```json
{
	"/endpoint": {
		"HTTP_METHOD": {
			"json_to_return_key": "json_to_return_value"
		}
	}
}
```

For example:

```json
{
	"/": {
		"GET": {
			"status": "200 OK"
		},
		"POST": {
			"herp": "derp"
		}
	},
	"/someendpoint": {
		"DELETE": {
			"deleted": 1
		}
	},
	"/some/other/endpoint": {
		"PUT": {
			"status": "ok"
		}
	}
}
```

# Limitations

Currently, `wayang` has no support for modifying the HTTP status code of the response value. This will be implemented once I can think of a sane way to encode this configuration within the mock-config JSON.

# What's Up With The Name?

Formally, wayang is a Javanese word for particular kinds of theatre.

Colloquially, [in Singapore](http://www.singlishdictionary.com/singlish_W.htm#wayang), wayang is a verb meaning "to put on a front", or "to pretend to be hard at work". It should now be immediately obvious why the name was chosen.
