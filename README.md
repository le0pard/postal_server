# Postal web server

This Docker image provides a web server that grants access to the [libpostal](https://github.com/openvenues/libpostal) library, enabling the parsing and normalization of street addresses globally. It need at least 2Gb of RAM (use 4GB for safety)

[Ready docker image](https://github.com/le0pard/postal_server/pkgs/container/postal_server)

```bash
docker pull ghcr.io/le0pard/postal_server:latest
```

## Usage

### Expand address

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

Support additional parameters:

- `languages`: An array of language codes (e.g., ["en", "fr"]) to help with expansion
- `address_components`: An array of strings specifying which parts of the address to expand. If not provided, a default set of components is used. See the **Address Components** section below for all available values.
- `latin_ascii`: Transliterate to Latin ASCII (`true` default value)
- `transliterate`: Transliterate to the script of the first language (`true` default value)
- `strip_accents`: Strip accents from the address (`true` default value)
- `decompose`: Decompose diacritics and other characters (`true` default value)
- `lowercase`: Convert the address to lowercase (`true` default value)
- `trim_string`: Trim leading and trailing whitespace (`true` default value)
- `replace_word_hyphens`: Replace hyphens in words with spaces (`true` default value)
- `delete_word_hyphens`: Delete hyphens in words (`true` default value)
- `replace_numeric_hyphens`: Replace hyphens in numbers with spaces (`false` default value)
- `delete_numeric_hyphens`: Delete hyphens in numbers (`false` default value)
- `split_alpha_from_numeric`: Split alphabetic and numeric parts of the address (`true` default value)
- `delete_final_periods`: Deletes final periods (`true` default value)
- `delete_acronym_periods`: Deletes periods in acronyms (e.g., "U.S.A." -> "USA") (`true` default value)
- `drop_english_possessives`: Drops "'s" from the end of tokens (e.g., "St. James's" -> "St. James") (`true` default value)
- `delete_apostrophes`: Deletes apostrophes (`true` default value)
- `expand_numex`: Expands numeric expressions (e.g., "Twenty-third" -> "23rd") (`true` default value)
- `roman_numerals`: Converts Roman numerals to integers (e.g., "II" -> "2") (`true` default value)

#### Address Components

You also can select which parts of the address to expand. If not provided, a default set of components is used

- `address_name`: The name of a venue, organization, or building
- `address_house_number`: The house or building number
- `address_street`: The street name
- `address_po_box`: Post office box numbers
- `address_unit`: An apartment, suite, or office number
- `address_level`: A floor or level number
- `address_entrance`: An entrance identifier, like "Lobby A"
- `address_staircase`: A staircase identifier
- `address_postal_code`: The postal code

### Parse address

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

Support additional parameters:

- `language`: The language of the address (e.g., "en")
- `country`: The country of the address (e.g., "us")

This will break down the address into its [individual components](https://github.com/openvenues/libpostal?tab=readme-ov-file#parser-labels).

### Healthcheck

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
POSTAL_SERVER_H2C - whether to use http2 h2c, default false
POSTAL_SERVER_DEBUG - enable debug mode, default false
```

## Development

Local build:

```bash
docker build -t postal-server .
```
