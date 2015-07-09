package main

import (
	"encoding/json"
        "flag"
	"fmt"
	"net/http"
	"time"
)

type synthPublApp struct {
	postEndpoint string
	s3          string
        uuid        string
}

type publication struct {
    succeeded   bool
    errorMsg    string
}

var postEndpoint = flag.String("postEndpoint","cms-notifier-pr-uk-int.svc.ft.com","publish endpoint address (most probably the address of cms-notifier in UCS)")
var tick = flag.Bool("tick", true, "periodially generate and post content to the post endpoint")
//fixed
var uuid = "01234567-89ab-cdef-0123-456789abcdef"

func main() {
	fmt.Printf("Starting synthetic image publication monitor...")

        flag.Parse()
        app := &synthPublApp{
            postEndpoint: *postEndpoint,
            uuid: uuid,
        }
        var _ = app

        bytes := make(chan []byte)
        lastResult := make(chan publication)
        if *tick {
            ticker := time.NewTicker(time.Second)
            go func() {
                    for _ = range ticker.C {
                        app.publish(bytes,lastResult)
                    }
            }()
        }
	http.HandleFunc("/__health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Healthcheck endpoint") })
	http.HandleFunc("/forcePublish", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "force publish") })
	http.HandleFunc("/test", testHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Could not start http server.")
	}
}

func (s *synthPublApp) publish(bytes chan<- []byte, history chan<- publication) {}

func testHandler(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(BuildRandomEOMImage(uuid))
	if err != nil {
		fmt.Fprintf(w, "Marshaling failed")
	}
	fmt.Fprintf(w, "test eom: \n%s", string(b))
}
