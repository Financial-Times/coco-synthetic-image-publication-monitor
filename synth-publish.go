package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/dchest/uniuri"
	"github.com/golang/go/src/pkg/encoding/base64"
)

const (
	stateCheckInterval = time.Duration(60) * time.Second
	postInterval       = time.Duration(120) * time.Second
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
		tick := time.Tick(postInterval)
		go func() {
			for {
				app.publish()
				<-tick
			}
		}()
	}
	go app.publishingMonitor()
	go app.historyManager()

	timedHC := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  "synth-image-pub-monitor",
			Name:        "Synthetic publication monitor",
			Description: "End-to-end image publication & monitor",
			Checks:      []fthealth.Check{app.healthcheck()},
		},
		Timeout: 10 * time.Second,
	}
	http.HandleFunc("/__health", fthealth.Handler(timedHC))
	http.HandleFunc(httphandlers.GTGPath, httphandlers.NewGoodToGoHandler(gtg.StatusChecker(app.GTG)))
	http.HandleFunc("/history", app.historyHandler)
	http.HandleFunc("/forcePublish", app.forcePublish)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Println("Error: Could not start http server.")
	}
}

func (app *syntheticPublication) GTG() gtg.Status {
	statusCheck := func() gtg.Status {
		return gtgCheck(app.latestPublicationStatus)
	}

	return gtg.FailFastParallelCheck([]gtg.StatusChecker{statusCheck})()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: "Image publication doesn't work"}
	}
	return gtg.Status{GoodToGo: true}
}

func (app *syntheticPublication) healthcheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Image publication doesn't work",
		Name:             "End-to-end test of image publication",
		PanicGuide:       "Contact #co-co channel on Slack",
		Severity:         1,
		TechnicalSummary: "Lots of things could have gone wrong. Check the /history endpoint for more info",
		Checker:          app.latestPublicationStatus,
	}
}

func (app *syntheticPublication) latestPublicationStatus() (string, error) {
	n := len(app.history)
	if n != 0 && !app.history[n-1].succeeded {
		msg := "Publication failed."
		return msg, errors.New(msg)
	}
	return "Ok", nil
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
	time.Sleep(stateCheckInterval)
	resp, err := http.Get(s3Endpoint)
	if err != nil {
		handlePublishingErr(result, latest.tid, latest.time, internalErr+"Executing Get request to s3 failed. "+err.Error())
		return
	}
	defer resp.Body.Close()
	StatusCode := http.StatusNotFound //resp.StatusCode
	switch StatusCode {
	case http.StatusOK:
		cmdR := exec.Command("kubectl", "delete", "cm", "synthetic-image-alarm", "--ignore-not-found")
		errR := cmdR.Run()
		if errR != nil {
			log.Fatalf("cmdR.Run() failed with %s\n", errR)
		}
	case http.StatusNotFound:
		handlePublishingErr(result, latest.tid, latest.time, "Image not found. Response status code: 404.")
		cmd1 := exec.Command("kubectl", "delete", "jobs", "image-trace-job", "--ignore-not-found")
		cmd2 := exec.Command("kubectl", "delete", "configmap", "synthetic-tid")
		cmd3 := exec.Command("kubectl", "create", "configmap", "synthetic-tid", "--from-literal=TID="+latest.tid)
		cmd4 := exec.Command("kubectl", "create", "job", "--from=cronjob/image-trace", "image-trace-job")
		cmdA := exec.Command("kubectl", "create", "cm", "synthetic-image-alarm", "--from-literal=alarm=true", "--dry-run=true", "-oyaml", "|", "kubectl", "apply", "-f", "-")

		err1 := cmd1.Run()
		if err1 != nil {
			log.Fatalf("cmd1.Run() failed with %s\n", err1)
		}
		err2 := cmd2.Run()
		if err2 != nil {
			log.Fatalf("cmd1.Run() failed with %s\n", err2)
		}
		err3 := cmd3.Run()
		if err3 != nil {
			log.Fatalf("cmd1.Run() failed with %s\n", err3)
		}
		err4 := cmd4.Run()
		if err4 != nil {
			log.Fatalf("cmd1.Run() failed with %s\n", err4)
		}
		errA := cmdA.Run()
		if errA != nil {
			log.Fatalf("cmdA.Run() failed with %s\n", errA)
		}
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
