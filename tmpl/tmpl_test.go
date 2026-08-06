package tmpl

import (
	"bytes"
	"strings"
	"testing"
)

func TestContains(t *testing.T) {
	if !Contains([]byte("{{.Email}}")) {
		t.Error("expected Contains to be true for templated input")
	}
	if Contains([]byte("no template here")) {
		t.Error("expected Contains to be false for plain input")
	}
}

func TestRenderPlainPassThrough(t *testing.T) {
	got, err := Render([]byte(`{"name":"fixed"}`))
	if err != nil {
		t.Fatalf("Render errored: %v", err)
	}
	if string(got) != `{"name":"fixed"}` {
		t.Errorf("got %q; want %q", got, `{"name":"fixed"}`)
	}
}

func TestRenderGoFields(t *testing.T) {
	tmpl, err := Parse([]byte(`{"email":"{{.Email}}","id":"{{.RequestID}}"}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	for i := 0; i < 50; i++ {
		got, err := tmpl.Render()
		if err != nil {
			t.Fatalf("Render errored: %v", err)
		}
		s := string(got)
		if !strings.Contains(s, "@") {
			t.Errorf("email field not filled: %s", s)
		}
		if strings.Contains(s, "{{") {
			t.Errorf("template markers left in output: %s", s)
		}
	}
}

func TestRenderBareMustacheFields(t *testing.T) {
	tmpl, err := Parse([]byte(`{"email":"{{email}}","name":"{{name}}","city":"{{city}}"}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	got, err := tmpl.Render()
	if err != nil {
		t.Fatalf("Render errored: %v", err)
	}
	s := string(got)
	if !strings.Contains(s, "@") {
		t.Errorf("email field not filled: %s", s)
	}
	if strings.Contains(s, "{{") {
		t.Errorf("template markers left in output: %s", s)
	}
}

func TestRenderGofakeitFuncs(t *testing.T) {
	tmpl, err := Parse([]byte(`{{Name}} {{FirstName}} {{City}} {{Phone}}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	for i := 0; i < 20; i++ {
		got, err := tmpl.Render()
		if err != nil {
			t.Fatalf("Render errored: %v", err)
		}
		if strings.Contains(string(got), "{{") {
			t.Errorf("template markers left in output: %s", got)
		}
	}
}

func TestRenderCaseInsensitiveFields(t *testing.T) {
	tmpl, err := Parse([]byte(`{{request_id}} {{DT_TM}} {{email_1}}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	got, err := tmpl.Render()
	if err != nil {
		t.Fatalf("Render errored: %v", err)
	}
	if strings.Contains(string(got), "{{") {
		t.Errorf("template markers left in output: %s", got)
	}
}

func TestRenderSuffixedFields(t *testing.T) {
	tmpl, err := Parse([]byte(`{"a":"{{.Email_1}}","b":"{{.Email_2}}"}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	got, err := tmpl.Render()
	if err != nil {
		t.Fatalf("Render errored: %v", err)
	}
	if strings.Contains(string(got), "{{") {
		t.Errorf("template markers left in output: %s", got)
	}
}

func TestRenderUnknownFieldErrors(t *testing.T) {
	_, err := Parse([]byte(`{{.Nope}}`))
	if err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestRenderGoFuncs(t *testing.T) {
	tmpl, err := Parse([]byte(`{{randInt 1 10}} {{Name}} {{randString 8}}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	for i := 0; i < 20; i++ {
		got, err := tmpl.Render()
		if err != nil {
			t.Fatalf("Render errored: %v", err)
		}
		if strings.Contains(string(got), "{{") {
			t.Errorf("template markers left in output: %s", got)
		}
	}
}

func TestRenderMustache(t *testing.T) {
	tmpl, err := Parse([]byte(`{"email":"{{email}}","id":"{{request_id}}"}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	for i := 0; i < 20; i++ {
		got, err := tmpl.Render()
		if err != nil {
			t.Fatalf("Render errored: %v", err)
		}
		s := string(got)
		if !strings.Contains(s, "@") {
			t.Errorf("email field not filled: %s", s)
		}
		if strings.Contains(s, "{{") {
			t.Errorf("template markers left in output: %s", s)
		}
	}
}

func TestRenderMustacheSection(t *testing.T) {
	tmpl, err := Parse([]byte(`hello {{#email}}{{email}}{{/email}}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	got, err := tmpl.Render()
	if err != nil {
		t.Fatalf("Render errored: %v", err)
	}
	if !strings.Contains(string(got), "@") {
		t.Errorf("section did not render: %s", got)
	}
}

func TestRenderMustacheComment(t *testing.T) {
	tmpl, err := Parse([]byte(`a{{! this is a comment }}b`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	got, err := tmpl.Render()
	if err != nil {
		t.Fatalf("Render errored: %v", err)
	}
	if string(got) != "ab" {
		t.Errorf("got %q; want %q", got, "ab")
	}
}

func TestRenderIsRandom(t *testing.T) {
	tmpl, err := Parse([]byte(`{{randInt 0 99999999}}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		got, err := tmpl.Render()
		if err != nil {
			t.Fatalf("Render errored: %v", err)
		}
		seen[string(got)] = true
	}
	if len(seen) < 2 {
		t.Error("expected randomized output across renders")
	}
}

func TestConcurrentRender(t *testing.T) {
	tmpl, err := Parse([]byte(`{"e":"{{.Email}}","n":{{randInt 1 100}},"s":"{{randString 8}}"}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	done := make(chan error, 32)
	for i := 0; i < 32; i++ {
		go func() {
			for j := 0; j < 200; j++ {
				got, err := tmpl.Render()
				if err != nil {
					done <- err
					return
				}
				if bytes.Contains(got, []byte("{{")) {
					done <- errUnfilled()
					return
				}
			}
			done <- nil
		}()
	}
	for i := 0; i < 32; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

type unfilledError struct{}

func (unfilledError) Error() string { return "template markers left in output" }

func errUnfilled() error { return unfilledError{} }

func TestVarHelperGo(t *testing.T) {
	SetValues(Vars{
		"user": {"alice", "bob", "charlie"},
		"city": {"Reykjavik", "Tokyo"},
	})
	tmpl, err := Parse([]byte(`{"user":"{{var "user"}}","city":"{{var "city"}}"}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		got, err := tmpl.Render()
		if err != nil {
			t.Fatalf("Render errored: %v", err)
		}
		s := string(got)
		if !strings.Contains(s, "alice") && !strings.Contains(s, "bob") && !strings.Contains(s, "charlie") {
			t.Errorf("value not picked from list: %s", s)
		}
		seen[s] = true
	}
	if len(seen) < 2 {
		t.Error("expected values to vary across renders")
	}
}

func TestVarHelperMustache(t *testing.T) {
	SetValues(Vars{"user": {"alice", "bob"}})
	tmpl, err := Parse([]byte(`{{! a comment }}{{var "user"}}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		got, err := tmpl.Render()
		if err != nil {
			t.Fatalf("Render errored: %v", err)
		}
		s := string(got)
		if s != "alice" && s != "bob" {
			t.Errorf("got %q; want alice or bob", s)
		}
		seen[s] = true
	}
	if len(seen) < 2 {
		t.Error("expected values to vary across renders")
	}
}

func TestVarHelperUnknownKey(t *testing.T) {
	SetValues(Vars{"known": {"a"}})
	if _, err := Parse([]byte(`{{var "missing"}}`)); err == nil {
		t.Error("expected error for unknown value key")
	}
}

func TestVarHelperConcurrent(t *testing.T) {
	SetValues(Vars{"user": {"alice", "bob", "charlie", "dave"}})
	tmpl, err := Parse([]byte(`{"user":"{{var "user"}}"}`))
	if err != nil {
		t.Fatalf("Parse errored: %v", err)
	}
	done := make(chan error, 16)
	for i := 0; i < 16; i++ {
		go func() {
			for j := 0; j < 200; j++ {
				if _, err := tmpl.Render(); err != nil {
					done <- err
					return
				}
			}
			done <- nil
		}()
	}
	for i := 0; i < 16; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}
