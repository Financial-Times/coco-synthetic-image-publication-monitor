package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	fthealth "github.com/Financial-Times/go-fthealth"
	"github.com/dchest/uniuri"
	"github.com/golang/go/src/pkg/encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type syntheticPublication struct {
	postEndpoint      string
	postCredentials   string
	s3Endpoint        string
	uuid              string
	latestImage       chan postedData
	latestPublication chan publicationResult

	mutex   *sync.Mutex
	history []publicationResult
}

type postedData struct {
	tid  string
	time time.Time
	img  string //base64 encoded string representation of the generated image
}

type publicationResult struct {
	tid       string
	time      time.Time
	succeeded bool
	errorMsg  string
}

var postHost = flag.String("postHost", "cms-notifier-pr-uk-int.svc.ft.com", "publish entrypoint host name (e.g. address of cms-notifier in UCS)")
var postCredentials = flag.String("postCredentials", "", "Authorization header value used to connect to the postHost")
var s3Host = flag.String("s3Host", "com.ft.imagepublish.int.s3.amazonaws.com", "saved image endpoint host name (e.g. address of the s3 service)")
var tick = flag.Bool("tick", true, "true, if this service should periodially generate and post content to the post endpoint")
var reqHeader = flag.Bool("dynRouting", false, "true, if post request is routed in a containerized environment through vulcan, therefore the request header must be set.")
var uuid = flag.String("testUuid", "c94a3a57-3c99-423c-a6bd-ed8c4c10a3c3", "uuid for the mock image used in the test")

func main() {
	log.Println("Starting synthetic image publication monitor...")

	flag.Parse()
	app := &syntheticPublication{
		postEndpoint:      buildPostEndpoint(*postHost),
		postCredentials:   *postCredentials,
		s3Endpoint:        buildGetEndpoint(*s3Host, *uuid),
		uuid:              *uuid,
		latestImage:       make(chan postedData),
		latestPublication: make(chan publicationResult),
		mutex:             &sync.Mutex{},
		history:           make([]publicationResult, 0),
	}

	if *tick {
		tick := time.Tick(time.Minute)
		go func() {
			for {
				app.publish()
				<-tick
			}
		}()
	}
	go app.publishingMonitor()
	go app.historyManager()

	http.HandleFunc("/__health", fthealth.Handler("Synthetic publication monitor", "End-to-end image publication & monitor", app.healthcheck()))
	http.HandleFunc("/history", app.historyHandler)
	http.HandleFunc("/forcePublish", app.forcePublish)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Println("Error: Could not start http server.")
	}
}

func (app *syntheticPublication) healthcheck() fthealth.Check {
	check := fthealth.Check{
		BusinessImpact:   "Image publication doesn't work",
		Name:             "End-to-end test",
		PanicGuide:       "Contact #co-co channel on Slack",
		Severity:         1,
		TechnicalSummary: "Lots of things could have gone wrong. Check the /history endpoint for more info",
		Checker:          app.latestPublicationStatus,
	}
	return check
}

func (app *syntheticPublication) latestPublicationStatus() error {
	n := len(app.history)
	if n != 0 && !app.history[n-1].succeeded {
		return errors.New("Publication failed.")
	}
	return nil
}

func (app *syntheticPublication) historyHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("History request.")
	app.mutex.Lock()
	for i := len(app.history) - 1; i >= 0; i-- {
		fmt.Fprintf(w, "%d. { Date: %s, Tid: %s, Succeeded: %t, Message: %s}\n\n",
			len(app.history)-i,
			app.history[i].time.String(),
			app.history[i].tid,
			app.history[i].succeeded,
			app.history[i].errorMsg,
		)
	}
	app.mutex.Unlock()
}

func (app *syntheticPublication) forcePublish(w http.ResponseWriter, r *http.Request) {
	log.Printf("Force publish.")
	err := app.publish()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Force publish failed. %s", err.Error())
	}
}

func (app *syntheticPublication) publish() error {
	eom, time := BuildRandomEOMImage(app.uuid)
	json, err := json.Marshal(eom)
	if err != nil {
		log.Printf("JSON marshalling failed. %s", err.Error())
		return err
	}
	buf := bytes.NewReader(json)

	client := http.Client{}
	req, err := http.NewRequest("POST", app.postEndpoint, buf)
	if err != nil {
		log.Printf("Error: Creating request failed. %s", err.Error())
		return err
	}
	tid := "SYNTHETIC-REQ-MON_" + uniuri.NewLen(10)
	req.Header.Add("X-Request-Id", tid)
	req.Header.Add("X-Origin-System-Id", "methode-web-pub")
	req.Header.Add("Authorization", app.postCredentials)
	if *reqHeader {
		req.Host = "cms-notifier"
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error: Executing request failed. %s", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		handlePublishingErr(app.latestPublication, tid, time, fmt.Sprintf("Publishing failed at first step: could not post data to CMS notifier. Status code: %d", resp.StatusCode))
	} else {
		app.latestImage <- postedData{tid, time, eom.Value}
	}

	return nil
}

const internalErr = "Internal error: "

func (app *syntheticPublication) publishingMonitor() {
	for latest := range app.latestImage {
		checkPublishingStatus(latest, app.latestPublication, app.s3Endpoint)
	}
}

func checkPublishingStatus(latest postedData, result chan<- publicationResult, s3Endpoint string) {
	sentImg, err := base64.StdEncoding.DecodeString(latest.img)
	if err != nil {
		handlePublishingErr(result, latest.tid, latest.time, internalErr+"Decoding image received from channed failed. "+err.Error())
		return
	}
	time.Sleep(30 * time.Second)
	resp, err := http.Get(s3Endpoint)
	if err != nil {
		handlePublishingErr(result, latest.tid, latest.time, internalErr+"Executing Get request to s3 failed. "+err.Error())
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		handlePublishingErr(result, latest.tid, latest.time, "Image not found. Response status code: 404.")
		return
	default:
		handlePublishingErr(result, latest.tid, latest.time, fmt.Sprintf("Get request is not successful. Response status code: %d", resp.StatusCode))
		return
	}

	receivedImg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handlePublishingErr(result, latest.tid, latest.time, internalErr+"Could not read resp body. "+err.Error())
		return
	}

	if !bytes.Equal(sentImg, receivedImg) {
		handlePublishingErr(result, latest.tid, latest.time, "Posted image content differs from the image in s3.")
		return
	}
	result <- publicationResult{latest.tid, latest.time, true, ""}
}

func handlePublishingErr(latestPublication chan<- publicationResult, tid string, time time.Time, errMsg string) {
	log.Printf("ERROR Publish failed. TID: " + tid + ". " + errMsg)
	latestPublication <- publicationResult{tid, time, false, errMsg}
}

func (app *syntheticPublication) historyManager() {
	for {
		latest := <-app.latestPublication

		app.mutex.Lock()
		if len(app.history) == 10 {
			app.history = app.history[1:len(app.history)]
		}
		app.history = append(app.history, latest)
		app.mutex.Unlock()
	}
}

func buildPostEndpoint(host string) string {
	return "http://" + host + "/notify"
}

func buildGetEndpoint(host, uuid string) string {
	return "http://" + host + "/" + uuid
}
