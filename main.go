package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/r3labs/sse/v2"
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

type AssetsHandler struct{}

func (h *AssetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	noCache(w)
	http.FileServer(http.FS(embeddedFS)).ServeHTTP(w, r)
}

func firstLabel(s string) string {
	if hostname, _, ok := strings.Cut(s, "."); ok {
		return hostname
	}

	return s
}

func main() {
	http.HandleFunc("/", home)
	http.HandleFunc("/set", set)
	http.HandleFunc("/get", get)
	http.Handle("/static/", &AssetsHandler{})

	sseServer.EncodeBase64 = true // if not done, only first line of multiline string will be send, see https://github.com/r3labs/sse/issues/62
	sseServer.AutoReplay = false
	sseServer.CreateStream("texts")
	http.HandleFunc("/events", sseServer.ServeHTTP)

	// generate cert with:
	// tailscale cert sd.alai-owl.ts.net
	baseDirForCerts := "" // fill it up
	crt := filepath.Join(baseDirForCerts, "device.tailnet-name.ts.net.crt")
	key := filepath.Join(baseDirForCerts, "device.tailnet-name.ts.net.key")
	log.Fatal(http.ListenAndServeTLS(":1111", crt, key, nil))
}
