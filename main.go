package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var text string

type HomeVars struct {
	Text string
}

func noCache(w http.ResponseWriter) {
	// from https://stackoverflow.com/a/5493543/7683365 and
	// https://web.dev/http-cache/#cache-control:~:text=browsers%20effectively%20guess%20what%20type%20of%20caching%20behavior%20makes%20the%20most%20sense
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
}

func home(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	noCache(w)
	if r.URL.Path != "/" {
		w.WriteHeader(404)
		return
	}

	t, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Template parsing error:", err)
		return
	}

	err = t.Execute(w, HomeVars{
		Text: text,
	})
}

func set(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	text = string(b)
}

func get(w http.ResponseWriter, r *http.Request) {
	noCache(w)
	w.Header().Add("Content-Type", "text/plain; charset=UTF-8")
	fmt.Fprint(w, text)
}

func typeSetter(w http.ResponseWriter, path string) func(contentType string, exts ...string) {
	return func(contentType string, exts ...string) {
		for _, ext := range exts {
			if strings.HasSuffix(path, ext) {
				w.Header().Set("Content-Type", contentType)
				return
			}
		}
	}
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}

	noCache(w)

	path := r.URL.Path
	/*
		Seems like urls are already resolved by browsers and curl before processing,
		so site.com/static/../secret-file becomes site.com/secret-file
		and hence is not handled by /static/ route.
		This means we don't get path traversal attacks.
		This however, is not applicable to query params and they are susceptible to it (either in url encoded form or w/o it)
		In case my assumption is incorrect, I would use https://pkg.go.dev/path/filepath#Clean
		and then retrieve the absolute path and then will make sure the resultant path starts with (program bin path + '/static/')
	*/
	data, err := os.ReadFile(path[1:])
	if err != nil {
		w.WriteHeader(404)
		return
	}

	setType := typeSetter(w, path)
	setType("text/javascript", ".js")
	setType("text/css", ".css")
	setType("image/svg+xml", ".svg", ".svgz")

	_, err = w.Write(data)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	http.HandleFunc("/", home)
	http.HandleFunc("/set", set)
	http.HandleFunc("/get", get)
	http.HandleFunc("/static/", staticHandler)
	// log.Fatal(http.ListenAndServe(":2222", nil))
	log.Fatal(http.ListenAndServeTLS(":1111", "../../sd.alai-owl.ts.net.crt", "../../sd.alai-owl.ts.net.key", nil))
}
