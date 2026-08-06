package tmpl

import "testing"

func BenchmarkRenderStatic(b *testing.B) {
	t, _ := Parse([]byte(`{"email":"fixed","n":1}`))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		t.Render()
	}
}

func BenchmarkRenderFields(b *testing.B) {
	t, _ := Parse([]byte(`{"email":"{{.Email}}","id":"{{.RequestID}}"}`))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		t.Render()
	}
}

func BenchmarkRenderHeavy(b *testing.B) {
	SetValues(Vars{"city": {"Reykjavik", "Tokyo", "New York"}})
	t, _ := Parse([]byte(`{"email":"{{.Email}}","name":"{{Name}}","city":"{{var "city"}}","age":{{randInt 18 90}},"uuid":"{{uuid}}"}`))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		t.Render()
	}
}
