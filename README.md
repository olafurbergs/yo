```
█████████                  ███████████                                                     █████████
█████████                 ████████████                                                     █████████
██████████░              █████████████░░                                                   █████████░░
██████████░              █████████████░░                 █                                 █████████░░
███████████░            ██████████████░░             █████████                             █████████░░
███████████░            █████████████░░░           █████████████                           █████████░░
████████████░          ██████████████░░░          ███████████████                          █████████░░
████████████░          █████████████░░░         ███████████████████                        █████████░░
█████████████░        ██████████████░░░         ███████████████████                        █████████░░
█████████████░       ██████████████░░░         █████████████████████░                      █████████░░
 █████████████░      █████████████░░░░        ████████░░░░░░░████████                      █████████░░
 █████████████░     ██████████████░░░        ████████░░░░░░░░░████████                     █████████░░
  █████████████░    █████████████░░░         ███████░░░░       ███████░                    █████████░░
  █████████████░   ██████████████░░░         ██████░░░░         ██████░░                   █████████░░
   █████████████░ ██████████████░░░         ███████░░░          ███████░                   █████████░░
   █████████████░ ██████████████░░░         ██████░░░            ██████░                   █████████░░
    ███████████████████████████░░░         ███████░░░            ███████░                  █████████░░
    ██████████████████████████░░░░         ██████░░░              ██████░                  █████████░░
     █████████████████████████░░░          ██████░░░              ██████░░                 █████████░░
     ████████████████████████░░░           ██████░░               ██████░░                 █████████░░
      ███████████████████████░░░           ██████░░               ██████░░                 █████████░░
      ██████████████████████░░░            ██████░░               ██████░░                 █████████░░
       ████████████████████░░░░            ██████░░               ██████░░                 █████████░░
       ████████████████████░░░            ███████░░               ███████░                 █████████░░
        ██████████████████░░░              ██████░░               ██████░░                 █████████░░
        ██████████████████░░░              ██████░░               ██████░░░                █████████░░
         ████████████████░░░               ██████░░               ██████░░                 █████████░░
         ████████████████░░░               ██████░░               ██████░░                 █████████░░
          ██████████████░░░                ██████░░               ██████░░                 █████████░░
          █████████████░░░░                ██████░░               ██████░░                 █████████░░
           ████████████░░░                 ███████░              ███████░░                 █████████░░
           ███████████░░░                   ██████░              ██████░░░                 █████████░░
           ███████████░░░                   ███████░            ███████░░░                 █████████░░
           ██████████░░░                     ██████░            ██████░░░                    ░░░░░░░░░
           █████████░░░░                     ███████░          ███████░░░                    ░░█░░░░░░
           █████████░░░                      ████████         ████████░░                    ███████
           █████████░░                        ████████       ████████░░░                   █████████
           █████████░░                         █████████████████████░░░░                  ███████████
           █████████░░                          ███████████████████░░░░                   ███████████░
           █████████░░                          ███████████████████░░░                    ███████████░░
           █████████░░                            ███████████████░░░░                    █████████████░
           █████████░░                            ░█████████████░░░░░                     ███████████░░
           █████████░░                              ░█████████░░░░░                       ███████████░░░
           █████████░░                               ░░░░█░░░░░░░░                        ███████████░░
           █████████░░                                 ░░░░░░░░░                           █████████░░░
           █████████░░                                     ░                                ███████░░░░
           █████████░░                                                                       ░░█░░░░░░
           █████████░░                                                                        ░░░░░░░
           █████████░░                                                                           ░
           █████████
```

yo is a tiny program that sends some load to a web application.

yo is a fork of the rakyll/hey HTTP load generator. It lets you randomize the URL,
headers and request body with Go or mustache templates
on every request, so a load test exercises many different inputs instead
of hammering the same request over and over.

## Installation

```
go install github.com/olafurbergs/yo
```

## Usage

yo runs provided number of requests in the provided concurrency level and prints stats.

It also supports HTTP2 endpoints.

```
Usage: yo [options...] <url>

Options:
  -n  Number of requests to run. Default is 200.
  -c  Number of workers to run concurrently. Total number of requests cannot
      be smaller than the concurrency level. Default is 50.
  -q  Rate limit, in queries per second (QPS) per worker. Default is no rate limit.
  -z  Duration of application to send requests. When duration is reached,
      application stops and exits. If duration is specified, n is ignored.
      Examples: -z 10s -z 3m.
  -s  Number of distinct request samples to generate. When set, yo
      pre-renders this many randomized requests at startup and cycles
      through them, so the target sees a bounded set of requests.
      Default is 0, meaning a fresh random request per load.
  -o  Output type. If none provided, a summary is printed.
      "csv" is the only supported alternative. Dumps the response
      metrics in comma-separated values format.

  -m  HTTP method, one of GET, POST, PUT, DELETE, HEAD, OPTIONS.
  -H  Custom HTTP header. You can specify as many as needed by repeating the flag.
      For example, -H "Accept: text/html" -H "Content-Type: application/xml" .
  -h  HTTP request headers as a template from a file.
  -t  Timeout for each request in seconds. Default is 20, use 0 for infinite.
  -A  HTTP Accept header.
  -d  HTTP request body.
  -D  HTTP request body from file.
  -b  HTTP request body as a template from a file.
  -p  HTTP request file. A full HTTP request in RFC 2616 format (request
      line, headers, an empty line and the body) that may contain templates.
      Overrides -m, -h, -d, -D and -b for the parts it specifies.
  -v  Values file. JSON or YAML mapping keys to arrays of values; every
      value must be an array. {{var "key"}} picks a random value from the
      array on each request.
  -T  Content-type, defaults to "text/html".
  -a  Basic authentication, username:password.
  -x  HTTP Proxy address as host:port.
  -h2 Enable HTTP/2.

  -host	HTTP Host header.

  -disable-compression  Disable compression.
  -disable-keepalive    Disable keep-alive, prevents re-use of TCP
                        connections between different HTTP requests.
  -disable-redirects    Disable following of HTTP redirects
  -cpus                 Number of used cpu cores.
                        (default for current machine is 8 cores)
```

## Templates

The URL, `-H` header values, `-h` header files, the `-d`/`-D`/`-b`
body and the `-p` request file may contain Go or mustache templates.
Every request re-renders them with fresh random values, so requests
vary throughout the run. The `-v` values file supplies named value
lists that `{{var "key"}}` picks from.

Both syntaxes are auto-detected and can be mixed with plain text:

* Go [text/template](https://pkg.go.dev/text/template) syntax.
* Mustache syntax: `{{var}}`, `{{{var}}}`, `{{#section}}...{{/section}}`,
  `{{^inverted}}...{{/inverted}}`, `{{! comment }}`.

### Random generators

yo bundles the full [gofakeit](https://github.com/brianvoe/gofakeit)
library, so every generator it ships is available inside a template.
Zero-argument generators that return a string can be used two ways:

* As a Go function call: `{{Name}}`, `{{Email}}`, `{{City}}`, `{{UUID}}`,
  `{{Phone}}`, `{{IPv4Address}}`, ...
* As a field — Go `{{.Email}}` or mustache `{{email}}` — because every
  zero-argument string generator is also registered as a ready-made field
  under the same name. Field names are matched case-insensitively and
  ignore underscores, so `{{.request_id}}` and `{{.RequestID}}` both work.

Generators that take arguments must be called Go-style with those
arguments, for example `{{Number 1 100}}`, `{{Sentence 10}}` or
`{{Password true true true true false 16}}`. A full list of generators
and their arguments is in the
[gofakeit docs](https://pkg.go.dev/github.com/brianvoe/gofakeit/v6).

#### Custom helpers

A few helpers yo adds on top of gofakeit, useful for shaping values:

* `{{randInt min max}}` — random integer between `min` and `max`.
* `{{randFloat min max}}` — random float between `min` and `max`.
* `{{randString n}}` — random alphanumeric string of length `n`.
* `{{uuid}}` — random UUID.
* `{{randBool}}` — random `true` or `false`.
* `{{choice "a" "b" "c"}}` — random one of the given values.

### Values file

The `-v` flag loads a values file — either JSON or YAML — whose keys map
to arrays of values. `{{var "key"}}` picks a random value from a key's
array on every request, without clashing with the random generators
above. Every value must be an array; yo fails if it is not.

```
cat values.json
{"username": ["alice", "bob", "charlie"], "city": ["Reykjavik", "Tokyo"]}
```

```
cat values.yaml
username:
  - alice
  - bob
  - charlie
city: [Reykjavik, Tokyo]
```

```
yo -n 100 -v values.json \
    'http://localhost:8080/{{var "city"}}/{{var "username"}}'
```

### Bounded request samples

By default every request is rendered fresh, so the target sees a
continuous stream of new requests. With `-s N`, yo pre-renders exactly
N distinct requests at startup and cycles through them round-robin,
bounding the population the target has to handle — useful for emulating
a fixed set of users or keeping results cache-friendly:

```
yo -n 1000 -c 50 -s 20 \
    -d '{"email":"{{.Email}}","age":{{randInt 18 90}}}' \
    http://localhost:8080/api/users
```

### Ready-made random fields

On top of the gofakeit generators above, yo ships a few curated fields
that always resolve as `{{.Field}}` (Go) or `{{field}}` (mustache):

* `{{.Email}}` — a random email address (same as gofakeit's `{{Email}}`).
* `{{.RequestID}}` — a random 32-char hex string (gofakeit UUID without
  dashes), handy for `X-Request-Id` headers.
* `{{.String}}` — a random 10-char alphanumeric string.
* `{{.Integer}}` — a random integer from 0 to 100000.
* `{{.Float}}` — a random float with 4 decimal places.
* `{{.Date}}`, `{{.Time}}`, `{{.DtTm}}` — random date, time of day, and
  datetime relative to now.

A numeric suffix (`{{.Email_1}}`, `{{.Email_2}}`) yields a separate,
independent value for each suffix — useful when a single request needs
two different emails. Because every zero-argument gofakeit generator is
also registered as a field (see above), `{{.City}}` works the same way.

## Examples

Make requests with default settings:

```
yo https://google.com
```

Make 1000 requests with 100 concurrent workers:

```
yo -n 1000 -c 100 https://google.com
```

Run load test for 30 seconds:

```
yo -z 30s https://google.com
```

Make POST request with custom body:

```
yo \
    -m POST \
    -d "param1=value1&param2=value2" \
    https://google.com
```

Add custom headers:

```
yo \
    -H "Accept: application/json" \
    -H "Authorization: Bearer token" \
    https://google.com
```

Test with HTTP/2:

```
yo -h2 https://google.com
```

Rate limit to 10 queries per second per worker:

```
yo -q 10 -c 5 -z 30s https://google.com
```

### Randomized requests

Randomize the URL path and query params:

```
yo -n 100 'https://api.example.com/users/{{.RequestID}}?city={{city}}'
```

Randomize the request body and a header on every POST:

```
yo -n 100 -m POST -T application/json \
    -H 'X-Request-Id: {{.RequestID}}' \
    -d '{"email":"{{.Email}}","name":"{{name}}","age":{{randInt 18 90}}}' \
    https://api.example.com/api/users
```

Keep templates in files with `-b` and `-h`:

```
cat body-template.yo
{"user_email": "{{.Email}}", "user_password": "{{randString 16}}"}

cat headers-template.yo
X-Request-Id: {{.RequestID}}
X-User-Agent-Hint: {{Name}}

yo -n 100 -m POST -T application/json -b body-template.yo -h headers-template.yo \
    'https://api.example.com/users/{{randInt 1 1000}}'
```

Set up the full request from an RFC 2616 request file with `-p`. The
method, request-target, headers and body are all templated per request:

```
cat request.http
POST /api/users HTTP/1.1
Host: {{.String}}.example.com
X-Request-Id: {{.RequestID}}
Content-Type: application/json

{"email": "{{.Email}}", "age": {{randInt 18 90}}}

yo -n 100 -p request.http https://api.example.com
```
