package main

import (
	"bytes"
	"encoding/base64"
	"math/rand"
	"text/template"
	"time"
)

type Eom struct {
	UUID             string        `json:"uuid"`
	EomType          string        `json:"type"`
	Value            string        `json:"value"`
	Attributes       string        `json:"attributes"`
	WorkflowStatus   string        `json:"workflowStatus"`
	SystemAttributes string        `json:"systemAttributes"`
	UsageTickets     string        `json:"usageTickets"`
	LinkedObjects    []interface{} `json:"linkedObjects"`
}

// Builds and populates an Eom struct matching the Methode EOM representation.
// The EOM value is a randomly generated 1000 length byte array encoded with base64, therefore producing a viewable string.
func BuildRandomEOMImage(uuid string) (*Eom, time.Time) {
	t := time.Now()

	eom := &Eom{}
	eom.UUID = uuid
	eom.EomType = "Image"
	eom.Value = base64.StdEncoding.EncodeToString(newByteArray(1000))
	eom.Attributes = populateTemplate("attributes.template", uuid)
	eom.WorkflowStatus = ""
	eom.SystemAttributes = populateTemplate("systemAttributes.template", t.Format("20060102"))
	eom.UsageTickets = populateTemplate("usageTickets.template", struct {
		UUID, Date, FormattedDate string
	}{
		uuid,
		t.Format("20060102150405"),
		t.Format(time.UnixDate),
	})
	eom.LinkedObjects = make([]interface{}, 0)
	return eom, t
}

func newByteArray(length int) []byte {
	b := make([]byte, length)
	for i := 0; i < len(b); i++ {
		b[i] = byte(rand.Intn(256))
	}
	return b
}

func populateTemplate(fileTempl string, data interface{}) string {
	tmpl, err := template.ParseFiles(fileTempl)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		panic(err)
	}
	return buf.String()
}
