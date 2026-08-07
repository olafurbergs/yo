// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command hey is an HTTP load generator.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	gourl "net/url"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/olafurbergs/yo/tmpl"
	"github.com/rakyll/hey/requester"
	"gopkg.in/yaml.v3"
)

const (
	headerRegexp = `^([\w-]+):\s*(.+)`
	authRegexp   = `^(.+):([^\s].+)`
)

// version is stamped at build time with
// -ldflags "-X main.version=v0.1.0" and defaults to "dev".
var version = "dev"

// defaultUA is appended to every request's User-Agent header.
var defaultUA = "yo/" + version

var (
	m            = flag.String("m", "GET", "")
	headers      = flag.String("h", "", "")
	body         = flag.String("d", "", "")
	bodyFile     = flag.String("D", "", "")
	bodyTemplate = flag.String("b", "", "")
	reqFile      = flag.String("p", "", "")
	valuesFile   = flag.String("v", "", "")
	accept       = flag.String("A", "", "")
	contentType  = flag.String("T", "text/html", "")
	authHeader   = flag.String("a", "", "")
	hostHeader   = flag.String("host", "", "")
	userAgent    = flag.String("U", "", "")

	output = flag.String("o", "", "")

	c = flag.Int("c", 50, "")
	n = flag.Int("n", 200, "")
	q = flag.Float64("q", 0, "")
	t = flag.Int("t", 20, "")
	z = flag.Duration("z", 0, "")
	s = flag.Int("s", 0, "")

	h2   = flag.Bool("h2", false, "")
	cpus = flag.Int("cpus", runtime.GOMAXPROCS(-1), "")

	disableCompression = flag.Bool("disable-compression", false, "")
	disableKeepAlives  = flag.Bool("disable-keepalive", false, "")
	disableRedirects   = flag.Bool("disable-redirects", false, "")
	proxyAddr          = flag.String("x", "", "")
)

var usage = `Usage: yo [options...] <url>

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
  -h  HTTP request headers as a template from a file. For example, /home/user/headers-template.yo or ./headers-template.yo
  -t  Timeout for each request in seconds. Default is 20, use 0 for infinite.
  -A  HTTP Accept header.
  -d  HTTP request body.
  -D  HTTP request body from file. For example, /home/user/file.txt or ./file.txt.
  -b  HTTP request body as a template from a file. For example, /home/user/body-template.yo or ./body-template.yo
  -p  HTTP request file. A full HTTP request in RFC 2616 format (request
      line, headers, an empty line and the body) that may contain templates.
      For example, /home/user/request.http or ./request.http. Overrides
      -m, -h, -d, -D and -b for the parts it specifies.
  -v  Values file. JSON or YAML mapping keys to arrays of values; every
      value must be an array. {{var "key"}} picks a random value from the
      array on each request. For example, /home/user/values.json or
      ./values.yaml
  -T  Content-type, defaults to "text/html".
  -U  User-Agent, defaults to "%s".
  -a  Basic authentication, username:password.
  -x  HTTP Proxy address as host:port.
  -h2 Enable HTTP/2.

  -host	HTTP Host header.

  -disable-compression  Disable compression.
  -disable-keepalive    Disable keep-alive, prevents re-use of TCP
                        connections between different HTTP requests.
  -disable-redirects    Disable following of HTTP redirects
  -cpus                 Number of used cpu cores.
                        (default for current machine is %d cores)

Templates:
  The URL, -H header values, -h header file, the -d/-D/-b body and the
  -p request file may contain Go or mustache templates. Every request
  renders them with fresh random values, so no two requests need to be
  identical.

  Mustache syntax (sections, inverted sections, comments and unescaped
  tags are supported) and Go text/template syntax are auto-detected and
  can be mixed with plain text.

  Every gofakeit generator is available as a Go template function, for
  example {{Name}}, {{Email}}, {{City}}, {{Phone}}, {{Number 1 100}}.
  Custom helpers: {{randInt min max}}, {{randFloat min max}},
  {{randString n}}, {{uuid}}, {{randBool}}, {{choice "a" "b" "c"}}.

  Ready-made random fields, usable in both Go ({{.Email}}) and mustache
  ({{email}}) forms. A numeric suffix such as Email_1 yields an
  independent value per request:
    {{.Email}}, {{.RequestID}}, {{.String}}, {{.Integer}}, {{.Float}},
    {{.Date}}, {{.Time}}, {{.DtTm}}.
  The -v values file provides named lists; {{var "key"}} picks a random
  value from the "key" list on every request, for example
  {{var "username"}}.
  Ex1: yo -n 100 -H 'X-Request-Id: {{.RequestID}}' http://localhost:8080/users
  Ex2: yo -n 100 -d '{"user_email": "{{.Email}}", "age": {{randInt 18 90}}}' \\
       -T application/json http://localhost:8080/api/users
  Ex3: yo -n 100 'http://localhost:8080/{{.String}}/{{randInt 1 100}}'
  Ex4: yo -n 100 -p request.http http://localhost:8080
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, defaultUA, runtime.NumCPU()))
	}

	var hs headerSlice
	flag.Var(&hs, "H", "")

	flag.Parse()

	var spec *parsedRequest
	if *reqFile != "" {
		data, err := os.ReadFile(*reqFile)
		if err != nil {
			errAndExit(err.Error())
		}
		spec, err = parseRequestFile(data)
		if err != nil {
			errAndExit(err.Error())
		}
	}

	if *valuesFile != "" {
		data, err := os.ReadFile(*valuesFile)
		if err != nil {
			errAndExit(err.Error())
		}
		vars, err := parseValuesFile(data)
		if err != nil {
			errAndExit(err.Error())
		}
		tmpl.SetValues(vars)
	}

	url := ""
	if flag.NArg() >= 1 {
		url = flag.Args()[0]
	}
	if spec != nil {
		if isAbsoluteTarget(spec.target) {
			url = spec.target
		} else {
			if url == "" {
				usageAndExit("a relative request-target in the request file requires a base <url> argument")
			}
			url = resolveTarget(url, spec.target)
		}
	} else if url == "" {
		usageAndExit("")
	}

	method := strings.ToUpper(*m)
	if spec != nil {
		method = spec.method
	}

	runtime.GOMAXPROCS(*cpus)
	num := *n
	conc := *c
	q := *q
	dur := *z

	if dur > 0 {
		num = math.MaxInt32
		if conc <= 0 {
			usageAndExit("-c cannot be smaller than 1.")
		}
	} else {
		if num <= 0 || conc <= 0 {
			usageAndExit("-n and -c cannot be smaller than 1.")
		}

		if num < conc {
			usageAndExit("-n cannot be less than -c.")
		}
	}

	if *s < 0 {
		usageAndExit("-s cannot be smaller than 0.")
	}

	// set content-type
	header := make(http.Header)
	header.Set("Content-Type", *contentType)
	// set any other additional headers
	if *headers != "" {
		slurp, err := os.ReadFile(*headers)
		if err != nil {
			errAndExit(err.Error())
		}
		for _, line := range strings.Split(string(slurp), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			hs = append(hs, line)
		}
	}
	// set any other additional repeatable headers
	for _, h := range hs {
		match, err := parseInputWithRegexp(h, headerRegexp)
		if err != nil {
			usageAndExit(err.Error())
		}
		header.Set(match[1], match[2])
	}

	if *accept != "" {
		header.Set("Accept", *accept)
	}

	// merge headers and host from the -p request file; the file wins.
	var hostValue string
	if spec != nil {
		if hv := spec.header.Get("Host"); hv != "" {
			hostValue = hv
			spec.header.Del("Host")
		}
		for k, vals := range spec.header {
			for _, v := range vals {
				header.Set(k, v)
			}
		}
	}

	// set basic auth if set
	var username, password string
	if *authHeader != "" {
		match, err := parseInputWithRegexp(*authHeader, authRegexp)
		if err != nil {
			usageAndExit(err.Error())
		}
		username, password = match[1], match[2]
	}

	var bodyAll []byte
	if spec != nil {
		bodyAll = spec.body
	} else {
		if *body != "" {
			bodyAll = []byte(*body)
		}
		if *bodyFile != "" {
			slurp, err := os.ReadFile(*bodyFile)
			if err != nil {
				errAndExit(err.Error())
			}
			bodyAll = slurp
		}
		if *bodyTemplate != "" {
			slurp, err := os.ReadFile(*bodyTemplate)
			if err != nil {
				errAndExit(err.Error())
			}
			bodyAll = slurp
		}
	}

	var proxyURL *gourl.URL
	if *proxyAddr != "" {
		var err error
		proxyURL, err = gourl.Parse(*proxyAddr)
		if err != nil {
			usageAndExit(err.Error())
		}
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		usageAndExit(err.Error())
	}
	req.ContentLength = int64(len(bodyAll))
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	// set host header if set
	if hostValue != "" {
		req.Host = hostValue
	}
	if *hostHeader != "" {
		req.Host = *hostHeader
	}

	ua := header.Get("User-Agent")
	if ua == "" {
		ua = defaultUA
	} else {
		ua += " " + defaultUA
	}
	header.Set("User-Agent", ua)

	// set userAgent header if set
	if *userAgent != "" {
		ua = *userAgent + " " + defaultUA
		header.Set("User-Agent", ua)
	}

	req.Header = header

	var reqTpl *requestTemplate
	if containsTemplate(url) || containsTemplateBytes(bodyAll) || headersContainTemplate(header) || (hostValue != "" && containsTemplate(hostValue)) {
		reqTpl, err = newRequestTemplate(url, bodyAll, header, hostValue)
		if err != nil {
			errAndExit(err.Error())
		}
	}

	w := &requester.Work{
		Request:            req,
		RequestBody:        bodyAll,
		N:                  num,
		C:                  conc,
		QPS:                q,
		Timeout:            *t,
		DisableCompression: *disableCompression,
		DisableKeepAlives:  *disableKeepAlives,
		DisableRedirects:   *disableRedirects,
		H2:                 *h2,
		ProxyAddr:          proxyURL,
		Output:             *output,
	}
	if reqTpl != nil {
		if *s > 0 {
			w.RequestFunc = reqTpl.sampler(req, *s)
		} else {
			w.RequestFunc = reqTpl.factory(req)
		}
	}
	w.Init()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		w.Stop()
	}()
	if dur > 0 {
		go func() {
			time.Sleep(dur)
			w.Stop()
		}()
	}
	w.Run()
}

// headerTemplate is a single header value that may contain template syntax.
type headerTemplate struct {
	key string
	tpl *tmpl.Template
}

// requestTemplate holds the compiled templates used to build a fresh,
// randomized request for every load request.
type requestTemplate struct {
	url   *tmpl.Template
	body  *tmpl.Template
	host  *tmpl.Template
	heads []headerTemplate
}

func containsTemplate(s string) bool {
	return tmpl.Contains([]byte(s))
}

func containsTemplateBytes(b []byte) bool {
	return tmpl.Contains(b)
}

func headersContainTemplate(h http.Header) bool {
	for _, vals := range h {
		for _, v := range vals {
			if tmpl.Contains([]byte(v)) {
				return true
			}
		}
	}
	return false
}

// newRequestTemplate compiles the URL, body, host and header templates. It
// returns an error if any of them uses unknown fields or invalid syntax.
func newRequestTemplate(url string, body []byte, header http.Header, host string) (*requestTemplate, error) {
	rt := &requestTemplate{}
	var err error
	rt.url, err = tmpl.Parse([]byte(url))
	if err != nil {
		return nil, err
	}
	rt.body, err = tmpl.Parse(body)
	if err != nil {
		return nil, err
	}
	if host != "" {
		rt.host, err = tmpl.Parse([]byte(host))
		if err != nil {
			return nil, err
		}
	}
	for key, vals := range header {
		for _, v := range vals {
			t, err := tmpl.Parse([]byte(v))
			if err != nil {
				return nil, err
			}
			rt.heads = append(rt.heads, headerTemplate{key: key, tpl: t})
		}
	}
	return rt, nil
}

// requestSample is a pre-rendered request variant.
type requestSample struct {
	url    *gourl.URL
	body   []byte
	host   string
	header http.Header
}

// renderSample renders one request variant with fresh random values.
func (rt *requestTemplate) renderSample() (requestSample, error) {
	u, err := rt.url.RenderString()
	if err != nil {
		return requestSample{}, err
	}
	parsed, err := gourl.Parse(u)
	if err != nil {
		return requestSample{}, err
	}
	body, err := rt.body.Render()
	if err != nil {
		return requestSample{}, err
	}
	var host string
	if rt.host != nil {
		host, err = rt.host.RenderString()
		if err != nil {
			return requestSample{}, err
		}
	}
	header := make(http.Header, len(rt.heads))
	for _, h := range rt.heads {
		v, err := h.tpl.RenderString()
		if err != nil {
			return requestSample{}, err
		}
		header.Add(h.key, v)
	}
	return requestSample{url: parsed, body: body, host: host, header: header}, nil
}

// materialize turns a request sample into a fresh, safe-to-use request.
func materialize(base *http.Request, s requestSample) *http.Request {
	r := base.Clone(context.Background())
	r.URL = s.url
	r.Body = ioutil.NopCloser(bytes.NewReader(s.body))
	r.ContentLength = int64(len(s.body))
	r.GetBody = func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader(s.body)), nil
	}
	r.Header = make(http.Header, len(s.header))
	for k, vals := range s.header {
		r.Header[k] = append([]string(nil), vals...)
	}
	if s.host != "" {
		r.Host = s.host
	}
	return r
}

// factory returns a function that builds a fresh request with randomized
// URL, body and headers for every call.
func (rt *requestTemplate) factory(base *http.Request) func() *http.Request {
	return func() *http.Request {
		s, err := rt.renderSample()
		if err != nil {
			errAndExit(err.Error())
		}
		return materialize(base, s)
	}
}

// sampler returns a function that pre-renders n distinct request variants
// and replays them one at a time, cycling round-robin. The target therefore
// sees a bounded set of n distinct requests.
func (rt *requestTemplate) sampler(base *http.Request, n int) func() *http.Request {
	samples := make([]requestSample, n)
	for i := range samples {
		s, err := rt.renderSample()
		if err != nil {
			errAndExit(err.Error())
		}
		samples[i] = s
	}
	var counter uint64
	return func() *http.Request {
		i := atomic.AddUint64(&counter, 1) % uint64(len(samples))
		return materialize(base, samples[i])
	}
}

func errAndExit(msg string) {
	fmt.Fprintf(os.Stderr, "%s", msg)
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func parseInputWithRegexp(input, regx string) ([]string, error) {
	re := regexp.MustCompile(regx)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 1 {
		return nil, fmt.Errorf("could not parse the provided input; input = %v", input)
	}
	return matches, nil
}

// parsedRequest holds the parts of an RFC 2616 request file passed via -p.
type parsedRequest struct {
	method string
	target string
	header http.Header
	body   []byte
}

// parseRequestFile parses an RFC 2616 request into its parts. The file is
// expected to contain a request line, headers and an empty line followed by
// the request body:
//
//	POST /users HTTP/1.1
//	Host: example.com
//	Content-Type: application/json
//
//	{"email": "{{.Email}}"}
//
// Header values, the request target and the body may contain templates.
func parseRequestFile(data []byte) (*parsedRequest, error) {
	head, body := splitHTTP(data)
	lines := strings.Split(head, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return nil, fmt.Errorf("request file is empty")
	}

	requestLine := strings.TrimSuffix(lines[0], "\r")
	i := strings.IndexByte(requestLine, ' ')
	if i < 1 {
		return nil, fmt.Errorf("malformed request line: %q", requestLine)
	}
	// The method is the first token. The request-target may itself contain
	// spaces when it is templated (e.g. /{{var "city"}}/x), so it is the
	// remainder of the line, minus a trailing HTTP version if present.
	target := strings.TrimSpace(requestLine[i+1:])
	if j := strings.LastIndex(target, " "); j > 0 && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(target[j+1:])), "HTTP/") {
		target = strings.TrimSpace(target[:j])
	}

	pr := &parsedRequest{
		method: strings.ToUpper(requestLine[:i]),
		target: target,
		header: make(http.Header),
		body:   body,
	}

	var lastKey string
	for _, line := range lines[1:] {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			if lastKey != "" {
				pr.header.Set(lastKey, pr.header.Get(lastKey)+" "+strings.TrimSpace(line))
			}
			continue
		}
		i := strings.IndexByte(line, ':')
		if i < 1 {
			return nil, fmt.Errorf("malformed header line: %q", line)
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		pr.header.Add(key, val)
		lastKey = key
	}
	return pr, nil
}

// splitHTTP splits an RFC 2616 message into its head and body sections.
func splitHTTP(data []byte) (head string, body []byte) {
	if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
		return string(data[:i]), data[i+4:]
	}
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return string(data[:i]), data[i+2:]
	}
	return string(data), nil
}

// isAbsoluteTarget reports whether a request-target is an absolute URL.
func isAbsoluteTarget(t string) bool {
	return strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://")
}

// resolveTarget joins a base URL with a relative request-target.
func resolveTarget(base, target string) string {
	if strings.HasSuffix(base, "/") && strings.HasPrefix(target, "/") {
		base = strings.TrimSuffix(base, "/")
	}
	return base + target
}

// parseValuesFile parses a values file into named value lists. The file
// may be either JSON or YAML; every value must be an array of scalars:
//
//	{"username": ["alice", "bob"], "city": ["Reykjavik", "Tokyo"]}
//
//	username:
//	  - alice
//	  - bob
//	city: [Reykjavik, Tokyo]
func parseValuesFile(data []byte) (tmpl.Vars, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err == nil {
		return valuesFromMap(raw)
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("values file is neither valid JSON nor YAML: %v", err)
	}
	return valuesFromMap(raw)
}

// valuesFromMap converts a decoded values document into value lists,
// failing if any value is not an array.
func valuesFromMap(raw map[string]interface{}) (tmpl.Vars, error) {
	vars := tmpl.Vars{}
	for k, v := range raw {
		list, ok := v.([]interface{})
		if !ok {
			return nil, fmt.Errorf("values file: key %q must be an array, got %T", k, v)
		}
		var vals []string
		for _, item := range list {
			s, err := valueString(item)
			if err != nil {
				return nil, fmt.Errorf("values file: key %q: %v", k, err)
			}
			vals = append(vals, s)
		}
		vars[k] = vals
	}
	return vars, nil
}

// valueString renders a decoded value as a template value. Scalars are
// converted to their string form; complex values are JSON-encoded.
func valueString(v interface{}) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", nil
	case string:
		return x, nil
	case bool:
		return strconv.FormatBool(x), nil
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(x), nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case uint64:
		return strconv.FormatUint(x, 10), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("unsupported value type %T", v)
		}
		return string(b), nil
	}
}

type headerSlice []string

func (h *headerSlice) String() string {
	return fmt.Sprintf("%s", *h)
}

func (h *headerSlice) Set(value string) error {
	*h = append(*h, value)
	return nil
}
