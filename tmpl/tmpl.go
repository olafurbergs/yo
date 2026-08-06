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

// Package tmpl renders Go and mustache templates that generate random
// values for every execution. It is used by yo to build requests whose
// URL, headers and body vary from call to call.
package tmpl

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"text/template/parse"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// Vars maps a name to the list of values {{var "name"}} picks from.
type Vars map[string][]string

var (
	valuesMu sync.RWMutex
	values   = map[string][]string{}
)

// SetValues sets the named value lists available to templates through the
// {{var "name"}} helper. It is safe to call before parsing templates.
func SetValues(v Vars) {
	valuesMu.Lock()
	defer valuesMu.Unlock()
	values = make(map[string][]string, len(v))
	for k, list := range v {
		values[k] = append([]string(nil), list...)
	}
}

// Template is a compiled yo template. Rendering it produces a fresh
// random document. Non-template inputs are rendered verbatim.
type Template struct {
	raw    string
	isTpl  bool
	goTpl  *template.Template
	fields []fieldSpec
}

// fieldSpec pairs a template field name with the generator that fills it.
type fieldSpec struct {
	name string
	gen  func() string
}

// Contains reports whether raw contains template syntax.
func Contains(raw []byte) bool {
	return bytes.Contains(raw, []byte("{{"))
}

// Parse compiles raw, auto-detecting mustache and Go template syntax.
func Parse(raw []byte) (*Template, error) {
	t := &Template{raw: string(raw)}
	if !Contains(raw) {
		return t, nil
	}
	t.isTpl = true
	s := t.raw
	if isMustache(s) {
		translated, err := translateMustache(s)
		if err != nil {
			return nil, err
		}
		s = translated
	}
	s = rewriteBareFields(s)
	goTpl, err := template.New("yo").Funcs(tmplFuncs).Parse(s)
	if err != nil {
		return nil, err
	}
	t.goTpl = goTpl
	fields := map[string]bool{}
	collectFields(goTpl.Tree.Root, fields)
	for name := range fields {
		gen, err := generatorFor(name)
		if err != nil {
			return nil, err
		}
		t.fields = append(t.fields, fieldSpec{name: name, gen: gen})
	}
	// Trial render to surface execution errors (e.g. calling a function
	// with the wrong number of arguments) before any load is sent.
	if _, err := t.Render(); err != nil {
		return nil, err
	}
	return t, nil
}

// Render executes the template and returns a fresh random document.
func (t *Template) Render() ([]byte, error) {
	if t == nil || !t.isTpl {
		if t == nil {
			return nil, nil
		}
		return []byte(t.raw), nil
	}
	data := make(map[string]interface{}, len(t.fields))
	for _, f := range t.fields {
		data[f.name] = f.gen()
	}
	var buf bytes.Buffer
	if err := t.goTpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderString is a convenience wrapper around Render.
func (t *Template) RenderString() (string, error) {
	b, err := t.Render()
	return string(b), err
}

// Render parses and renders raw in a single call.
func Render(raw []byte) ([]byte, error) {
	t, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	return t.Render()
}

// RenderString parses and renders s in a single call.
func RenderString(s string) (string, error) {
	b, err := Render([]byte(s))
	return string(b), err
}

// isMustache reports whether s uses mustache-specific constructs.
func isMustache(s string) bool {
	return strings.Contains(s, "{{#") ||
		strings.Contains(s, "{{^") ||
		strings.Contains(s, "{{&") ||
		strings.Contains(s, "{{!") ||
		strings.Contains(s, "{{{") ||
		(strings.Contains(s, "{{/") && !strings.Contains(s, "{{/*"))
}

// goKeywords are reserved words that must not be rewritten into field
// lookups by rewriteBareFields.
var goKeywords = map[string]bool{
	"if": true, "range": true, "with": true, "end": true, "else": true,
	"define": true, "template": true, "block": true, "break": true,
	"continue": true, "nil": true, "true": true, "false": true,
}

// bareTagRe matches a single bare {{ identifier }} tag.
var bareTagRe = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// rewriteBareFields rewrites bare {{ name }} tags that are neither Go
// keywords nor registered functions into {{ .name }} field lookups. This
// lets plain mustache-style interpolation ({{ email }}, {{ name }}) work
// alongside Go function calls ({{ Name }}, {{ randInt 1 10 }}).
func rewriteBareFields(s string) string {
	return bareTagRe.ReplaceAllStringFunc(s, func(tag string) string {
		m := bareTagRe.FindStringSubmatch(tag)
		name := m[1]
		if goKeywords[name] {
			return tag
		}
		if _, ok := tmplFuncs[name]; ok {
			return tag
		}
		return "{{." + name + "}}"
	})
}

// mustacheRe matches a single mustache tag, triple-brace tags first.
var mustacheRe = regexp.MustCompile(`(?s)\{\{\{.*?\}\}\}|\{\{.*?\}\}`)

// translateMustache rewrites mustache syntax into Go template syntax so
// it can be executed by text/template. Mustache variables resolve against
// the same generated field data as Go templates.
func translateMustache(s string) (string, error) {
	var err error
	out := mustacheRe.ReplaceAllStringFunc(s, func(tag string) string {
		if err != nil {
			return tag
		}
		var inner string
		if strings.HasPrefix(tag, "{{{") {
			inner = strings.TrimSpace(tag[3 : len(tag)-3])
		} else {
			inner = strings.TrimSpace(tag[2 : len(tag)-2])
		}
		switch {
		case inner == "":
			return ""
		case strings.HasPrefix(inner, "."):
			// Already a Go-style field (e.g. {{.}}).
			return "{{" + inner + "}}"
		case strings.HasPrefix(inner, "#"):
			return "{{if ." + strings.TrimSpace(inner[1:]) + "}}"
		case strings.HasPrefix(inner, "^"):
			return "{{if not ." + strings.TrimSpace(inner[1:]) + "}}"
		case strings.HasPrefix(inner, "/"):
			return "{{end}}"
		case strings.HasPrefix(inner, "&"):
			return "{{." + strings.TrimSpace(inner[1:]) + "}}"
		case strings.HasPrefix(inner, "!"):
			return ""
		case strings.HasPrefix(inner, "="):
			err = fmt.Errorf("mustache delimiter changes are not supported")
			return tag
		default:
			// Function calls (e.g. {{var "name"}}, {{randInt 1 10}}) are
			// already valid Go template syntax and are passed through.
			if words := strings.Fields(inner); len(words) > 0 {
				if _, ok := tmplFuncs[words[0]]; ok {
					return "{{" + inner + "}}"
				}
			}
			return "{{." + inner + "}}"
		}
	})
	if err != nil {
		return "", err
	}
	return out, nil
}

// collectFields records every field node referenced by the template so a
// value can be generated for it before execution.
func collectFields(node parse.Node, fields map[string]bool) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *parse.ActionNode:
		collectPipe(n.Pipe, fields)
	case *parse.PipeNode:
		collectPipe(n, fields)
	case *parse.ListNode:
		for _, child := range n.Nodes {
			collectFields(child, fields)
		}
	case *parse.IfNode:
		collectBranch(&n.BranchNode, fields)
	case *parse.RangeNode:
		collectBranch(&n.BranchNode, fields)
	case *parse.WithNode:
		collectBranch(&n.BranchNode, fields)
	case *parse.TemplateNode:
		collectPipe(n.Pipe, fields)
	}
}

func collectBranch(bn *parse.BranchNode, fields map[string]bool) {
	collectPipe(bn.Pipe, fields)
	if bn.List != nil {
		collectFields(bn.List, fields)
	}
	if bn.ElseList != nil {
		collectFields(bn.ElseList, fields)
	}
}

func collectPipe(p *parse.PipeNode, fields map[string]bool) {
	if p == nil {
		return
	}
	for _, cmd := range p.Cmds {
		for _, arg := range cmd.Args {
			if f, ok := arg.(*parse.FieldNode); ok {
				fields[strings.Join(f.Ident, ".")] = true
			}
		}
	}
}

// fieldGenerators maps the well-known hey-style field names to their
// random value generators. Every zero-argument gofakeit generator that
// returns a string is registered too (see registerFieldFuncs).
var fieldGenerators = map[string]func() string{
	"Email":     func() string { return gofakeit.Email() },
	"RequestID": func() string { return strings.ReplaceAll(gofakeit.UUID(), "-", "") },
	"String":    func() string { return randString(10) },
	"Integer":   func() string { return strconv.Itoa(gofakeit.Number(0, 100000)) },
	"Float":     func() string { return strconv.FormatFloat(gofakeit.Float64Range(0, 1000), 'f', 4, 64) },
	"Date":      randomDate,
	"Time":      randomTime,
	"DtTm":      randomDateTime,
}

func init() {
	for name, f := range tmplFuncs {
		fv := reflect.ValueOf(f)
		if fv.Kind() != reflect.Func || fv.Type().NumIn() != 0 ||
			fv.Type().NumOut() != 1 || fv.Type().Out(0).Kind() != reflect.String {
			continue
		}
		if _, exists := fieldGenerators[name]; !exists {
			fieldGenerators[name] = fv.Interface().(func() string)
		}
	}
}

// normalize lower-cases s and drops underscores so spellings such as
// request_id and RequestID match the same generator.
func normalize(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", ""))
}

// generatorFor resolves a field name to a generator. Field names may
// carry a numeric suffix (e.g. Email_1, Email_2) so a template can ask
// for several independent values of the same kind.
func generatorFor(field string) (func() string, error) {
	if gen, ok := fieldGenerators[field]; ok {
		return gen, nil
	}
	nField := normalize(field)
	for name, gen := range fieldGenerators {
		if normalize(name) == nField {
			return gen, nil
		}
	}
	if i := strings.Index(field, "_"); i > 0 {
		nBase := normalize(field[:i])
		for name, gen := range fieldGenerators {
			if normalize(name) == nBase {
				return gen, nil
			}
		}
	}
	return nil, fmt.Errorf("unknown template field %q", field)
}

// tmplFuncs exposes every gofakeit generator plus yo's custom helpers to
// Go templates. It is built once; gofakeit's locked source makes it safe
// for concurrent use by the load workers.
var tmplFuncs = buildTemplateFuncs()

// buildTemplateFuncs registers all gofakeit Faker methods as template
// functions (mirroring gofakeit's own template engine) and then adds yo's
// custom helpers.
func buildTemplateFuncs() template.FuncMap {
	fm := template.FuncMap{}
	f := gofakeit.New(0)
	v := reflect.ValueOf(f)
	excluded := map[string]bool{
		"RandomMapKey": true,
		"SQL":          true,
		"Template":     true,
	}
	for i := 0; i < v.NumMethod(); i++ {
		name := v.Type().Method(i).Name
		if excluded[name] {
			continue
		}
		if v.Type().Method(i).Type.NumOut() == 0 {
			continue
		}
		fm[name] = v.Method(i).Interface()
	}
	fm["randInt"] = func(min, max int) int { return gofakeit.Number(min, max) }
	fm["randFloat"] = func(min, max float64) float64 { return gofakeit.Float64Range(min, max) }
	fm["randString"] = randString
	fm["uuid"] = func() string { return gofakeit.UUID() }
	fm["randBool"] = func() bool { return gofakeit.Bool() }
	fm["choice"] = func(items ...interface{}) interface{} {
		if len(items) == 0 {
			return nil
		}
		return items[gofakeit.Number(0, len(items)-1)]
	}
	fm["var"] = func(name string) (string, error) {
		valuesMu.RLock()
		defer valuesMu.RUnlock()
		list, ok := values[name]
		if !ok || len(list) == 0 {
			return "", fmt.Errorf("template value %q is not defined", name)
		}
		return list[gofakeit.Number(0, len(list)-1)], nil
	}
	return fm
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randString(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[gofakeit.Number(0, len(letterBytes)-1)]
	}
	return string(b)
}

func randomDate() string {
	return time.Now().UTC().AddDate(0, 0, gofakeit.Number(-365, 365)).Format("2006-01-02")
}

func randomTime() string {
	return time.Now().UTC().Add(time.Duration(gofakeit.Number(0, 86399)) * time.Second).Format("15:04:05")
}

func randomDateTime() string {
	now := time.Now().UTC().AddDate(0, 0, gofakeit.Number(-365, 365))
	return now.Add(time.Duration(gofakeit.Number(0, 86399)) * time.Second).Format("2006-01-02 15:04:05")
}
