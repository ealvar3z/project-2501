//go:build ignore

package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	addr := "localhost:8000"
	if len(os.Args) >= 2 && os.Args[1] == "-a" {
		addr = "localhost:0"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ln.Addr().(*net.TCPAddr).Port)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stop" {
			_, _ = w.Write([]byte{})
			go func() { _ = ln.Close() }()
			return
		}
		if r.URL.Path == "/headers" {
			for k, vals := range r.Header {
				for _, v := range vals {
					fmt.Fprintf(w, "%s: %s\n", strings.ToLower(k), v)
				}
			}
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		data, err := os.ReadFile(name)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if strings.HasSuffix(name, ".http") {
			head, body, _ := strings.Cut(string(data), "\n\n")
			for _, line := range strings.Split(head, "\n") {
				k, v, ok := strings.Cut(line, ":")
				if ok {
					w.Header().Add(k, strings.TrimSpace(v))
				}
			}
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write(data)
	})}
	if err := server.Serve(ln); err != nil && err != net.ErrClosed {
		log.Fatal(err)
	}
}
