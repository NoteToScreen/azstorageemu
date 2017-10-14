package main

import (
	"encoding/xml"
	"net/http"
)

type BlobEnumerationResults struct {
	XMLName    xml.Name     `xml:"EnumerationResults"`
	Prefix     string       `xml:"Prefix"`
	Marker     string       `xml:"Marker"`
	MaxResults int64        `xml:"MaxResults"`
	Delimiter  string       `xml:"Delimiter"`
	Blobs      []BlobResult `xml:"Blobs>Blob"`
	NextMarker string       `xml:"NextMarker"`
}

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
