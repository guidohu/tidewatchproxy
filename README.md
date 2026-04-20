# Tide Watch Proxy

A lightweight Go-based proxy server that sits between the Garmin watch face ([Tide Watch](https://github.com/guidohu/tidewatch)) and the [Stormglass.io](https://stormglass.io) and [BigDataCloud](https://www.bigdatacloud.com/) APIs.

It fetches data from the Stormglass API, filters for requested parameters, and returns a dense JSON payload. This reduces the memory footprint of the API response, ensuring it comfortably fits within the strict 32KB memory constraints of a Garmin watch background service.

## Features

- **Dense Data Format**: Minimizes JSON keys and removes unnecessary metadata for Garmin memory efficiency.
- **Weather Point Data**: Proxies swell height, period, and direction (supports `noaa` source).
- **Tide Extremes**: Retrieves high and low tide times and heights.
- **Sea Level Data**: Retrieves time-series sea level data with custom datum support (e.g., `MLLW`).
- **Reverse Geocoding**: Integrated with BigDataCloud and supports **custom location overrides** via CSV.
- **Redis Caching**: Caches responses locally to stay within API rate limits and speed up subsequent requests.
- **Strict Validation**: Enforces input parameters and filters requested weather parameters.
- **Access Control**: Secures proxy access with allowed App IDs.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)
- A Stormglass.io API key if you want to serve users that don't have to provide a key.

## Running Locally

1. Create a `custom_locations.csv` file if you want to override geocoding results (format: `lat,lon,name`).
2. Update the `STORMGLASS_API_KEY` in `docker-compose.yml`.
3. Run the stack:

```bash
docker-compose up -d --build
```

The application will run on `http://localhost:8080`.

## Endpoints

### Weather Point
`GET /v2/weather/point?lat=...&lng=...&params=...&source=noaa`
- **Params**: `swellHeight`, `swellPeriod`, `swellDirection`, `secondarySwellHeight`, `secondarySwellPeriod`, `secondarySwellDirection`.
- **Default Source**: `noaa`.

**Example Request**:
```bash
curl "http://localhost:8080/v2/weather/point?lat=21.27&lng=-157.82&params=swellHeight,swellPeriod,swellDirection&source=noaa"
```

**Example Response**:
```json
{
  "data": [
    {
      "ts": 1776640237,
      "h1": 1.45,
      "d1": 185.2,
      "p1": 14.1
    }
  ]
}
```

### Tide Extremes
`GET /v2/tide/extremes/point?lat=...&lng=...&start=...&end=...`

**Example Request**:
```bash
curl "http://localhost:8080/v2/tide/extremes/point?lat=21.27&lng=-157.82&start=1776640237&end=1776726637"
```

**Example Response**:
```json
{
  "data": [
    {
      "ts": 1776640237,
      "h": 0.45
    },
    {
      "ts": 1776661837,
      "h": 1.2
    }
  ]
}
```

### Sea Level
`GET /v2/tide/sea-level/point?lat=...&lng=...&start=...&end=...&datum=MLLW`

**Example Request**:
```bash
curl "http://localhost:8080/v2/tide/sea-level/point?lat=21.27&lng=-157.82&start=1776640237&end=1776643837&datum=MLLW"
```

**Example Response**:
```json
{
  "data": [
    {
      "ts": 1776640237,
      "h": 0.45
    },
    {
      "ts": 1776643837,
      "h": 0.52
    }
  ]
}
```

### Reverse Geocode
`GET /data/reverse-geocode-client?latitude=...&longitude=...`
- Returns local name/city. Checks `custom_locations.csv` first.

**Example Request**:
```bash
curl "http://localhost:8080/data/reverse-geocode-client?latitude=21.27&longitude=-157.82"
```

**Example Response**:
```json
{
  "locality": "Waikiki",
  "city": "Honolulu",
  "countryName": "United States",
  "countryCode": "US"
}
```

## Environment Variables

- `STORMGLASS_API_KEY`: Your Stormglass.io API key.
- `REDIS_ADDR`: Address of the Redis instance (default: `redis:6379`).
- `PORT`: Port to listen on (default: `8080`).
- `ALLOWED_APP_IDS`: Restrict proxy access to a comma-separated list of App IDs. Clients must pass this as `X-App-Id` headers or `app_id` query parameters. Leave blank to allow any request.

## Development

Run natively for development:
```bash
go run main.go --use-cache=false --custom-locations=./custom_locations.csv
```
