# Postal web server

Docker image with webserver, which provide access to [libpostal](https://github.com/openvenues/libpostal) library. Can be used for for parsing/normalizing street addresses around the world.

[Ready docker image](https://github.com/le0pard/postal_server/pkgs/container/postal_server)

## Usage

To expand address strings into normalized forms suitable for geocoder queries you need to use `/expand` endpoint with `address` query parameter. Example to expand "Quatre-vingt-douze Ave des Ave des Champs-Élysées" address:

```bash
GET /expand?address=Quatre-vingt-douze%20Ave%20des%20Ave%20des%20Champs-Élysées

[
  "92 avenue des avenue des champs-elysees",
  "92 avenue des avenue des champs elysees",
  "92 avenue des avenue des champselysees"
]
```

To parse addresses into components you need to use `/parse` endpoint with `address` query parameter. Example to parse "781 Franklin Ave Crown Heights Brooklyn NY 11216 USA" address:

```bash
GET /expand?address=Quatre-vingt-douze%20Ave%20des%20Ave%20des%20Champs-Élysées

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

Endpoint `/health` can be use to check webserver healthcheck (like in k8s env):

```bash
$ curl http://localhost:8000/health
{"status":"ok"}
```

## Configuration

Configuration environment variables:

```ini
POSTAL_SERVER_HOST - server host (default: 0.0.0.0)
POSTAL_SERVER_PORT - server port (default: 8000)
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
# x86 linux or mac os
docker build -t postal-server .
# mac os with apple silicon
docker build -t postal-server --build-arg LIBPOSTAL_CONFIGURE_FLAGS=--disable-sse2 .
```
