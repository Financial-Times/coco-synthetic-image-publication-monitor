package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	_ "time"
)

type endToEndTest struct {
	cmsNotifier string
	s3          string
}

//fixed
var uuid = "8fe6ee49-cbd9-4f3e-8e7d-202164cf4eb5"

func main() {
	fmt.Printf("Starting synthetic image publication monitor...")
	//ticker := time.NewTicker(time.Second)
	go func() {
		//for t := range ticker.C {
		//    fmt.Println("Tick at", t)
		//}
	}()
	http.HandleFunc("/__health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Healthcheck endpoint") })
	http.HandleFunc("/forcePublish", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "force publish") })
	http.HandleFunc("/test", testHandler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Could not start http server.")
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(BuildRandomEOMImage(uuid))
	if err != nil {
		fmt.Fprintf(w, "Marshaling failed")
	}
	fmt.Fprintf(w, "test eom: \n%s", string(b))
}
