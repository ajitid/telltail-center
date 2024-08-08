package main

import (
	"bytes"
	"crypto/tls"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/r3labs/sse/v2"
	"tailscale.com/tsnet"
)

// replace with "github.com/urfave/cli/v2" if this gets serious
var (
	client = &http.Client{}

	pushoverUser  = flag.String("pushover-user", "", "")
	pushoverToken = flag.String("pushover-token", "", "")
	// optional
	pushoverDevice = flag.String("pushover-device", "", "")
)

type pushoverData struct {
	User     string `json:"user"`
	Token    string `json:"token"`
	Priority int8   `json:"priority"`
	Ttl      uint32 `json:"ttl"`
	Message  string `json:"message"`
	Device   string `json:"device,omitempty"`
}

// ----

var (
	text      string      // should be wrapped in a mutex?
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

	if len(*pushoverUser) > 0 && len(*pushoverToken) > 0 {
		payload, err := json.Marshal(&pushoverData{
			User:     *pushoverUser,
			Token:    *pushoverToken,
			Priority: -2,
			Ttl:      1,
			Message:  p.Text,
			Device:   *pushoverDevice,
		})
		if err != nil {
			// TODO add log (not fatal)
			return
		}

		req, err := http.NewRequest("POST", "https://api.pushover.net/1/messages.json", bytes.NewBuffer(payload))
		if err != nil {
			// TODO add log (not fatal)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			// TODO add log (not fatal)
			return
		}
		defer resp.Body.Close()
	}
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
	flag.Parse()

	if len(os.Getenv("TS_AUTHKEY")) == 0 {
		log.Fatal("`TS_AUTHKEY` environment variable is not set")
	}

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
