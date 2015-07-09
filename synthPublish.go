package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dchest/uniuri"
	"log"
	"net/http"
	"sync"
	"time"
)

type syntheticPublication struct {
	postHost string
	s3Host   string
	uuid     string
	//base64 encoded string representation of the generated image
	latestImage       chan string
	latestPublication chan publication

	mutex   *sync.Mutex
	history []publication
}

type publication struct {
	succeeded bool
	errorMsg  string
}

var postHost = flag.String("postHost", "cms-notifier-pr-uk-int.svc.ft.com", "publish entrypoint host name (e.g. address of cms-notifier in UCS)")
var s3Host = flag.String("s3Host", "com.ft.imagepublish.int.s3.amazonaws.com", "saved image endpoint host name (e.g. address of the s3 host)")
var tick = flag.Bool("tick", true, "true, if this service should periodially generate and post content to the post endpoint")

//fixed
var uuid = "c94a3a57-3c99-423c-a6bd-ed8c4c10a3c3"

func main() {
	log.Println("Starting synthetic image publication monitor...")

	flag.Parse()
	app := &syntheticPublication{
		postHost:          *postHost,
		s3Host:            *s3Host,
		uuid:              uuid,
		latestImage:       make(chan string),
		latestPublication: make(chan publication),
		mutex:             &sync.Mutex{},
		history:           make([]publication, 10),
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
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Println("Error: Could not start http server.")
	}
}

func (app *syntheticPublication) forcePublish(w http.ResponseWriter, r *http.Request) {
	log.Printf("Force publish.")
	err := app.publish()
	if err != nil {
		fmt.Fprintf(w, "Force publish failed. %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *syntheticPublication) publish() error {
	eom := BuildRandomEOMImage(uuid)
	json, err := json.Marshal(eom)
	if err != nil {
		log.Printf("JSON marshalling failed. %s", err.Error())
		return err
	}
	buf := bytes.NewReader(json)

	client := http.Client{}
	req, err := http.NewRequest("POST", buildPostEndpoint(app.postHost), buf)
	if err != nil {
		log.Printf("Error: Creating request failed. %s", err.Error())
		return err
	}
	req.Header.Add("X-Request-Id", "SYNTHETIC-REQ-MON_"+uniuri.NewLen(10))
	req.Header.Add("X-Origin-System-Id", "methode-web-pub")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error: Executing request failed. %s", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errMsg := fmt.Sprintf("Publishing failed at first step: could not post data to CMS notifier. Status code: %d", resp.StatusCode)
		app.latestPublication <- publication{false, errMsg}
	} else {
		app.latestImage <- eom.Value
	}

	return nil
}

func buildPostEndpoint(host string) string {
	return "http://" + host + "/notify"
}
