# Tide Watch Proxy

A lightweight Go-based proxy server that sits between the Garmin watch face ([Tide Watch](https://github.com/guidohu/tidewatch)) and the Surfline API.

It fetches data from the Surfline API, strips away unnecessary fields, and returns a minimized JSON payload. This reduces the memory footprint of the API response, ensuring it comfortably fits within the strict 32KB memory constraints of a Garmin watch background service.

## Features

- Proxies `/kbyg/mapview` (spot lookup and mapping).
- Proxies `/kbyg/spots/forecasts/tides` (tide data).
- Proxies `/kbyg/spots/forecasts/wave` (swell and wave data).
- Caches responses in Redis to avoid hitting Surfline rate limits and speed up responses.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

## Running Locally

The easiest way to run the proxy is using Docker Compose, which will spin up both the Go proxy application and the Redis cache.

```bash
docker-compose up -d --build
```

The application will run on `http://localhost:8080`.

## Example Usage

**Spot Lookup (Mapview)**
```bash
curl "http://localhost:8080/kbyg/mapview?lat=21.64&lon=-158.06&distance=10"
```

**Tide Forecast**
```bash
curl "http://localhost:8080/kbyg/spots/forecasts/tides?spotId=5842041f4e65fad6a7708890&days=2"
```

**Wave Forecast**
```bash
curl "http://localhost:8080/kbyg/spots/forecasts/wave?spotId=5842041f4e65fad6a7708890&days=2"
```

## Development

If you prefer to run the application natively (without Docker for the Go app):

1. Start a Redis instance locally or update `REDIS_ADDR` in your environment.
2. Run the application:

```bash
go run main.go
```

By default it listens on `:8080` unless the `PORT` environment variable is set. Cache can be disabled by passing the `-use-cache=false` flag.
