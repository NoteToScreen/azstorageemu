package main

import (
	"encoding/xml"
	"net/http"
)

type BlobErrorResponse struct {
	Code    string
	Message string
}

type BlobBlockListResponse struct {
	Blocks []string `xml:"Uncommitted"`
}

func writeXML(w http.ResponseWriter, thing interface{}) error {
	data, err := xml.Marshal(thing)
	if err != nil {
		return err
	}
	w.Write(data)
	return nil
}
