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

package main

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/olafurbergs/yo/tmpl"
)

func TestParseValidHeaderFlag(t *testing.T) {
	match, err := parseInputWithRegexp("X-Something: !Y10K:;(He@poverflow?)", headerRegexp)
	if err != nil {
		t.Errorf("parseInputWithRegexp errored: %v", err)
	}
	if got, want := match[1], "X-Something"; got != want {
		t.Errorf("got %v; want %v", got, want)
	}
	if got, want := match[2], "!Y10K:;(He@poverflow?)"; got != want {
		t.Errorf("got %v; want %v", got, want)
	}
}

func TestParseInvalidHeaderFlag(t *testing.T) {
	_, err := parseInputWithRegexp("X|oh|bad-input: badbadbad", headerRegexp)
	if err == nil {
		t.Errorf("Header parsing errored; want no errors")
	}
}

func TestParseValidAuthFlag(t *testing.T) {
	match, err := parseInputWithRegexp("_coo-kie_:!!bigmonster@1969sid", authRegexp)
	if err != nil {
		t.Errorf("A valid auth flag was not parsed correctly: %v", err)
	}
	if got, want := match[1], "_coo-kie_"; got != want {
		t.Errorf("got %v; want %v", got, want)
	}
	if got, want := match[2], "!!bigmonster@1969sid"; got != want {
		t.Errorf("got %v; want %v", got, want)
	}
}

func TestParseInvalidAuthFlag(t *testing.T) {
	_, err := parseInputWithRegexp("X|oh|bad-input: badbadbad", authRegexp)
	if err == nil {
		t.Errorf("Header parsing errored; want no errors")
	}
}

func TestParseAuthMetaCharacters(t *testing.T) {
	_, err := parseInputWithRegexp("plus+$*{:boom", authRegexp)
	if err != nil {
		t.Errorf("Auth header with a plus sign in the user name errored: %v", err)
	}
}

func TestHeadersContainTemplate(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", "text/html")
	if headersContainTemplate(h) {
		t.Error("expected no template detection for static headers")
	}
	h.Set("X-Request-Id", "{{.RequestID}}")
	if !headersContainTemplate(h) {
		t.Error("expected template detection for templated header")
	}
}

func TestRequestTemplateFactory(t *testing.T) {
	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set("X-Request-Id", "{{.RequestID}}")

	base, err := http.NewRequest("POST", "http://example.com/{{randString 4}}", nil)
	if err != nil {
		t.Fatalf("NewRequest errored: %v", err)
	}
	base.Host = "example.com"
	base.Header = header

	rt, err := newRequestTemplate("http://example.com/{{randString 4}}/{{.Integer}}", []byte(`{"email":"{{.Email}}"}`), header, "")
	if err != nil {
		t.Fatalf("newRequestTemplate errored: %v", err)
	}

	factory := rt.factory(base)
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		r := factory()
		if r.Method != "POST" {
			t.Errorf("got method %q; want POST", r.Method)
		}
		if r.Host != "example.com" {
			t.Errorf("got host %q; want example.com", r.Host)
		}
		if r.URL.Path == "" || strings.Contains(r.URL.Path, "{{") {
			t.Errorf("URL not templated: %v", r.URL)
		}
		if rid := r.Header.Get("X-Request-Id"); rid == "" || strings.Contains(rid, "{{") {
			t.Errorf("X-Request-Id not templated: %q", rid)
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading body errored: %v", err)
		}
		if !strings.Contains(string(body), "@") || strings.Contains(string(body), "{{") {
			t.Errorf("body not templated: %s", body)
		}
		seen[r.URL.Path] = true
	}
	if len(seen) < 2 {
		t.Error("expected randomized URLs across requests")
	}
}

func TestRequestTemplateSampler(t *testing.T) {
	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set("X-Request-Id", "{{.RequestID}}")

	base, err := http.NewRequest("POST", "http://example.com/{{.RequestID}}", nil)
	if err != nil {
		t.Fatalf("NewRequest errored: %v", err)
	}
	base.Host = "example.com"
	base.Header = header

	rt, err := newRequestTemplate("http://example.com/{{.RequestID}}", []byte(`{"id":"{{.RequestID}}"}`), header, "")
	if err != nil {
		t.Fatalf("newRequestTemplate errored: %v", err)
	}

	gen := rt.sampler(base, 5)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		r := gen()
		if r.Method != "POST" {
			t.Errorf("got method %q; want POST", r.Method)
		}
		if r.Host != "example.com" {
			t.Errorf("got host %q; want example.com", r.Host)
		}
		if strings.Contains(r.URL.Path, "{{") || strings.Contains(r.Header.Get("X-Request-Id"), "{{") {
			t.Fatalf("request not rendered: %v %v", r.URL, r.Header)
		}
		seen[r.URL.Path] = true
	}
	if got := len(seen); got != 5 {
		t.Errorf("expected exactly 5 distinct sampled requests, got %d", got)
	}
}

func TestRequestTemplateSamplerBodyIndependent(t *testing.T) {
	header := http.Header{}
	header.Set("Content-Type", "application/json")

	base, err := http.NewRequest("POST", "http://example.com/{{.RequestID}}", nil)
	if err != nil {
		t.Fatalf("NewRequest errored: %v", err)
	}
	base.Header = header

	rt, err := newRequestTemplate("http://example.com/{{.RequestID}}", []byte(`{"id":"{{.RequestID}}"}`), header, "")
	if err != nil {
		t.Fatalf("newRequestTemplate errored: %v", err)
	}

	gen := rt.sampler(base, 3)
	// Concurrent calls must never share a mutable body reader.
	done := make(chan bool, 16)
	for i := 0; i < 16; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				r := gen()
				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Errorf("reading body errored: %v", err)
					continue
				}
				if strings.Contains(string(b), "{{") {
					t.Errorf("body not rendered: %s", b)
				}
			}
			done <- true
		}()
	}
	for i := 0; i < 16; i++ {
		<-done
	}
}

func TestParseRequestFile(t *testing.T) {
	data := []byte("POST /users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"X-Fold: a\r\n" +
		"   b\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		`{"email": "{{.Email}}", "id": {{.RequestID}}}` + "\n")

	spec, err := parseRequestFile(data)
	if err != nil {
		t.Fatalf("parseRequestFile errored: %v", err)
	}
	if got, want := spec.method, "POST"; got != want {
		t.Errorf("method = %q; want %q", got, want)
	}
	if got, want := spec.target, "/users"; got != want {
		t.Errorf("target = %q; want %q", got, want)
	}
	if got, want := spec.header.Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q; want %q", got, want)
	}
	if got, want := spec.header.Get("X-Fold"), "a b"; got != want {
		t.Errorf("folded X-Fold = %q; want %q", got, want)
	}
	if got, want := string(spec.body), `{"email": "{{.Email}}", "id": {{.RequestID}}}`+"\n"; got != want {
		t.Errorf("body = %q; want %q", got, want)
	}
}

func TestParseRequestFileNoBody(t *testing.T) {
	spec, err := parseRequestFile([]byte("GET /healthz HTTP/1.1\nHost: example.com\n\n"))
	if err != nil {
		t.Fatalf("parseRequestFile errored: %v", err)
	}
	if spec.method != "GET" || spec.target != "/healthz" {
		t.Errorf("got %s %s; want GET /healthz", spec.method, spec.target)
	}
	if len(spec.body) != 0 {
		t.Errorf("body = %q; want empty", spec.body)
	}
}

func TestParseRequestFileMalformed(t *testing.T) {
	if _, err := parseRequestFile([]byte("not-a-request-line\n")); err == nil {
		t.Error("expected error for malformed request line")
	}
	if _, err := parseRequestFile([]byte("GET / HTTP/1.1\nThisIsNotAHeader\n\n")); err == nil {
		t.Error("expected error for malformed header line")
	}
}

func TestResolveTarget(t *testing.T) {
	cases := []struct{ base, target, want string }{
		{"http://localhost:8080", "/users", "http://localhost:8080/users"},
		{"http://localhost:8080/", "/users", "http://localhost:8080/users"},
		{"http://localhost:8080/api", "/users", "http://localhost:8080/api/users"},
	}
	for _, c := range cases {
		if got := resolveTarget(c.base, c.target); got != c.want {
			t.Errorf("resolveTarget(%q, %q) = %q; want %q", c.base, c.target, got, c.want)
		}
	}
	if isAbsoluteTarget("/users") {
		t.Error("expected /users to be relative")
	}
	if !isAbsoluteTarget("https://example.com/v1") {
		t.Error("expected https://example.com/v1 to be absolute")
	}
}

func TestParseValuesFileJSON(t *testing.T) {
	data := []byte(`{
		"username": ["alice", "bob", "charlie"],
		"city": ["Reykjavik", "Tokyo"],
		"port": [8080, 9090],
		"enabled": [true, false]
	}`)
	vars, err := parseValuesFile(data)
	if err != nil {
		t.Fatalf("parseValuesFile errored: %v", err)
	}
	want := tmpl.Vars{
		"username": {"alice", "bob", "charlie"},
		"city":     {"Reykjavik", "Tokyo"},
		"port":     {"8080", "9090"},
		"enabled":  {"true", "false"},
	}
	if len(vars) != len(want) {
		t.Fatalf("got %d keys; want %d", len(vars), len(want))
	}
	for k, v := range want {
		got, ok := vars[k]
		if !ok {
			t.Errorf("key %q missing", k)
			continue
		}
		if strings.Join(got, ",") != strings.Join(v, ",") {
			t.Errorf("values[%q] = %v; want %v", k, got, v)
		}
	}
}

func TestParseValuesFileYAML(t *testing.T) {
	data := []byte("# comment\n" +
		"username:\n" +
		"  - alice\n" +
		"  - bob\n" +
		"city: [Reykjavik, Tokyo]\n")
	vars, err := parseValuesFile(data)
	if err != nil {
		t.Fatalf("parseValuesFile errored: %v", err)
	}
	want := tmpl.Vars{
		"username": {"alice", "bob"},
		"city":     {"Reykjavik", "Tokyo"},
	}
	if len(vars) != len(want) {
		t.Fatalf("got %d keys; want %d", len(vars), len(want))
	}
	for k, v := range want {
		got, ok := vars[k]
		if !ok {
			t.Errorf("key %q missing", k)
			continue
		}
		if strings.Join(got, ",") != strings.Join(v, ",") {
			t.Errorf("values[%q] = %v; want %v", k, got, v)
		}
	}
}

func TestParseValuesFileNonArrayFails(t *testing.T) {
	if _, err := parseValuesFile([]byte(`{"key": "not-an-array"}`)); err == nil {
		t.Error("expected error for JSON non-array value")
	}
	if _, err := parseValuesFile([]byte("key: scalar\n")); err == nil {
		t.Error("expected error for YAML non-array value")
	}
}

func TestParseValuesFileInvalid(t *testing.T) {
	if _, err := parseValuesFile([]byte("this is : not valid: yaml [ or json")); err == nil {
		t.Error("expected error for invalid values file")
	}
}
