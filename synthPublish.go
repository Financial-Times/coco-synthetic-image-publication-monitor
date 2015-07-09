package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
        "sync"
	"time"
)

type syntheticPublication struct {
	postEndpoint        string
	s3                  string
	uuid                string
        latestImage         chan []byte
        latestPublication   chan publication

        mutex               *sync.Mutex
        history             []publication
}

type publication struct {
	succeeded bool
	errorMsg  string
}

var postEndpoint = flag.String("postEndpoint", "cms-notifier-pr-uk-int.svc.ft.com", "publish endpoint address (most probably the address of cms-notifier in UCS)")
var tick = flag.Bool("tick", true, "true, if this service should periodially generate and post content to the post endpoint")

//fixed
var uuid = "01234567-89ab-cdef-0123-456789abcdef"

func main() {
        log.Println("Starting synthetic image publication monitor...")

	flag.Parse()
	app := &syntheticPublication{
		postEndpoint: *postEndpoint,
		uuid:         uuid,
                latestImage: make(chan []byte),
                latestPublication: make(chan publication),
                mutex:       &sync.Mutex{},
                history:     make([]publication, 10),
	}

	if *tick {
		ticker := time.NewTicker(time.Second)
		go func() {
			for _ = range ticker.C {
				app.publish()
			}
		}()
	}
	http.HandleFunc("/__health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Healthcheck endpoint") })
	http.HandleFunc("/forcePublish", app.forcePublish)
	http.HandleFunc("/test", testHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Println("Could not start http server.")
	}
}

func (app *syntheticPublication) forcePublish(w http.ResponseWriter, r *http.Request) {
	log.Printf("Force publish.")
        app.publish()
}

func (app *syntheticPublication) publish() {
	b, err := json.Marshal(BuildRandomEOMImage(uuid))
	if err != nil {
		log.Println("JSON marshalling failed.")
		return
	}
	buf := bytes.NewReader(b)
	resp, err := http.Post(buildPostEndpoint(app.postEndpoint), "application/json; charset=utf-8", buf)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
                errMsg := fmt.Sprintf("Publishing failed at first step: could not post data to CMS notifier. Status code: %d", resp.StatusCode);
		app.latestPublication <- publication{false, errMsg}
	} else {
		app.latestImage <- b
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(BuildRandomEOMImage(uuid))
	if err != nil {
		fmt.Fprintf(w, "Marshaling failed")
	}
	fmt.Fprintf(w, "test eom: \n%s", string(b))
}

func buildPostEndpoint(host string) string {
    return "http://" + host + "/notify"
}
