package main

import (
	"bytes"
	"math/rand"
	"text/template"
	"time"
)

type Eom struct {
	Uuid             string        `json:"uuid"`
	EomType          string        `json:"type"`
	Value            string        `json:"value"`
	Attributes       string        `json:"attributes"`
	WorkflowStatus   string        `json:"workflowStatus"`
	SystemAttributes string        `json:"systemAttributes"`
	UsageTickets     string        `json:"usageTickets"`
	LinkedObjects    []interface{} `json:"linkedObjects"`
}

func BuildRandomEOMImage(uuid string) *Eom {
	t := time.Now()

	eom := &Eom{}
	eom.Uuid = uuid
	eom.EomType = "Image"
	eom.Value = string(newByteArray(1000))
	eom.Attributes = populateTemplate("attributes.template", uuid)
	eom.WorkflowStatus = ""
	eom.SystemAttributes = populateTemplate("systemAttributes.template", t.Format("20060102"))
	eom.UsageTickets = populateTemplate("usageTickets.template", struct {
		Uuid, Date, FormattedDate string
	}{
		uuid,
		t.Format("20060102150405"),
		t.Format(time.UnixDate),
	})
	eom.LinkedObjects = make([]interface{}, 0)
	return eom
}

func newByteArray(length int) []byte {
	b := make([]byte, length)
	for i := 0; i < len(b); i++ {
		b[i] = byte(rand.Intn(128-48) + 48)
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
