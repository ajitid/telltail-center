package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
)

var text string

type HomeVars struct {
	Text string
}

func home(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Print("Template parsing error:", err)
	}

	err = t.Execute(w, HomeVars{
		Text: text,
	})
}

func set(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}
	text = string(b)
}

func get(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=UTF-8")
	fmt.Fprint(w, text)
}

func main() {
	http.HandleFunc("/", home)
	http.HandleFunc("/set", set)
	http.HandleFunc("/get", get)
	// log.Fatal(http.ListenAndServe(":1111", nil))
	log.Fatal(http.ListenAndServeTLS(":1111", "../../sd.alai-owl.ts.net.crt", "../../sd.alai-owl.ts.net.key", nil))
}
