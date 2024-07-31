# Postal web server

This Docker image provides a web server that grants access to the [libpostal](https://github.com/openvenues/libpostal) library, enabling the parsing and normalization of street addresses globally.

[Ready docker image](https://github.com/le0pard/postal_server/pkgs/container/postal_server)

```bash
docker pull ghcr.io/le0pard/postal_server:latest
```

## Usage

To expand address strings into normalized forms suitable for geocoder queries, use the `/expand` endpoint with the `address` query parameter. For example, to expand the address "Quatre-vingt-douze Ave des Ave des Champs-Élysées":

```bash
GET /expand?address=Quatre-vingt-douze%20Ave%20des%20Ave%20des%20Champs-Élysées

[
  "92 avenue des avenue des champs-elysees",
  "92 avenue des avenue des champs elysees",
  "92 avenue des avenue des champselysees"
]
```

This will provide the expanded and normalized addresses ready for geocoding queries.

To parse addresses into components, use the `/parse` endpoint with the `address` query parameter. For example, to parse the address "781 Franklin Ave Crown Heights Brooklyn NY 11216 USA":

```bash
GET /parse?address=781%20Franklin%20Ave%20Crown%20Heights%20Brooklyn%20NY%2011216%20USA

[
  {
    "label": "house_number",
    "value": "781"
  },
  {
    "label": "road",
    "value": "franklin ave"
  },
  {
    "label": "suburb",
    "value": "crown heights"
  },
  {
    "label": "city_district",
    "value": "brooklyn"
  },
  {
    "label": "state",
    "value": "ny"
  },
  {
    "label": "postcode",
    "value": "11216"
  },
  {
    "label": "country",
    "value": "usa"
  }
]
```

This will break down the address into its [individual components](https://github.com/openvenues/libpostal?tab=readme-ov-file#parser-labels).

Endpoint `/health` can be use to check webserver healthcheck (like in k8s env):

```bash
$ curl http://localhost:8000/health
{"status":"ok"}
```

## Auth for server

You can set up either basic authentication or bearer token authentication to protect your web server, while keeping the `/health` endpoint public

## Configuration

Configuration environment variables:

```ini
POSTAL_SERVER_HOST - server host (default: 0.0.0.0)
POSTAL_SERVER_PORT or PORT - server port (default: 8000)
POSTAL_SERVER_TRUSTED_PROXIES - trusted proxies IP addresses (separated by comma)
POSTAL_SERVER_LOG_FORMAT - log format, can be "json" or "text" (default: "text")
POSTAL_SERVER_LOG_LEVEL - log level (default: "info")
POSTAL_SERVER_BASIC_AUTH_USERNAME - basic auth username (required if basic auth password is set)
POSTAL_SERVER_BASIC_AUTH_PASSWORD - basic auth password (required if basic auth username is set)
POSTAL_SERVER_BEARER_AUTH_TOKEN - bearer auth token
```

## Development

Local build:

```bash
docker build -t postal-server .
```
