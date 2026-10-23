package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	address := flag.String("listen", "127.0.0.1:8787", "HTTP listen address")
	username := flag.String("username", "plex", "Basic Auth username")
	password := flag.String("password", "demo", "Basic Auth password")
	flag.Parse()

	handler := demoHandler(*username, *password)
	log.Printf("PlexLinker demo server: http://%s/", *address)
	log.Printf("username=%s password=%s", *username, *password)
	log.Fatal(http.ListenAndServe(*address, handler))
}

func demoHandler(username, password string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", listing(`<a href="Shows/"><strong>Shows/</strong></a><br><a href="Movies/">Movies/</a>`))
	mux.HandleFunc("/Shows/", listing(`<a href="../">Parent Directory</a><table><tr><td><a href="Example.Show/">Example Show/</a></td><td>2026-10-23 10:00</td></tr></table>`))
	mux.HandleFunc("/Shows/Example.Show/", listing(`<a href="../">Parent Directory</a><table><tr><td><a href="Example.Show.S01E01.Pilot.mkv"><span>Example.Show.S01E01.Pilot.mkv</span></a></td><td>2026-10-23 10:01</td></tr><tr><td><a href="Example.Show.S01E02.Next.mkv">Example.Show.S01E02.Next.mkv</a></td><td>2026-10-23 10:02</td></tr></table>`))
	mux.HandleFunc("/Movies/", listing(`<a href="../">Parent Directory</a><pre><a href="Example.Movie.2026.mp4">Example.Movie.2026.mp4</a> 23-Oct-2026 10:03</pre>`))
	for _, path := range []string{"/Shows/Example.Show/Example.Show.S01E01.Pilot.mkv", "/Shows/Example.Show/Example.Show.S01E02.Next.mkv", "/Movies/Example.Movie.2026.mp4"} {
		path := path
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			payload := sha256.Sum256([]byte(path))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, path[strings.LastIndex(path, "/")+1:]))
			_, _ = w.Write(payload[:])
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="PlexLinker demo"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func listing(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<!doctype html><html><body>%s</body></html>", body)
	}
}
