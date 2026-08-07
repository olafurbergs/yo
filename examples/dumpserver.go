// Command dumpserver prints every request it receives, so you can see
// what yo generates. Run it in one terminal:
//
//	go run ./examples/dumpserver.go
//
// then point yo at http://localhost:8080 from another.
package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fmt.Printf("%s %s\n", r.Method, r.URL.RequestURI())
		for k, vs := range r.Header {
			for _, v := range vs {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}
		if len(body) > 0 {
			fmt.Printf("  BODY: %s\n", body)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	fmt.Println("listening on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
