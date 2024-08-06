package main

import (
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/r3labs/sse/v2"
	"tailscale.com/tsnet"
)

var (
	text      string
	sseServer *sse.Server = sse.New()
)

//go:embed static index.html
var embeddedFS embed.FS

type homeTemplateVars struct {
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

	t, err := template.ParseFS(embeddedFS, "index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Println("Template parsing error:", err)
		return
	}

	err = t.Execute(w, homeTemplateVars{
		Text: text,
	})
}

type payload struct {
	Text   string
	Device string
}

func set(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(400)
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	var p payload
	json.Unmarshal(b, &p)
	if p.Device == "" {
		// must specify the device as `unknown` if there is hestitancy in uniquely naming it
		w.WriteHeader(400)
		return
	}
	// 65536 is not an arbitrarily picked number, see https://www.wikiwand.com/en/65,536#/In_computing
	if len(p.Text) == 0 || len(p.Text) > 65536 {
		return
	}
	text = p.Text
	sseServer.Publish("texts", &sse.Event{
		Data: b,
	})
}

func get(w http.ResponseWriter, r *http.Request) {
	noCache(w)
	w.Header().Add("Content-Type", "text/plain; charset=UTF-8")
	fmt.Fprint(w, text)
}

type assetsHandler struct{}

func (h *assetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	noCache(w)
	http.FileServer(http.FS(embeddedFS)).ServeHTTP(w, r)
}

func main() {
	os.Setenv("TSNET_FORCE_LOGIN", "1")

	s := &tsnet.Server{
		Hostname: "telltail",
	}
	defer s.Close()

	ln, err := s.Listen("tcp", ":443")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	lc, err := s.LocalClient()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", home)
	http.HandleFunc("/set", set)
	http.HandleFunc("/get", get)
	http.Handle("/static/", &assetsHandler{})

	sseServer.EncodeBase64 = true // if not done, only first line of multiline string will be send, see https://github.com/r3labs/sse/issues/62
	sseServer.AutoReplay = false
	sseServer.CreateStream("texts")
	http.HandleFunc("/events", sseServer.ServeHTTP)

	// serve with http:
	// log.Fatal(http.Serve(ln, nil))
	// or with https:
	server := http.Server{
		TLSConfig: &tls.Config{
			GetCertificate: lc.GetCertificate,
		},
	}
	log.Fatal(server.ServeTLS(ln, "", ""))
}
