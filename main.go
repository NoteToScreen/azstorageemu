package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
)

var config Config

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	log.Println("azstorageemu")

	flag.IntVar(&config.BlobStorage.Port, "blob-port", 10000, "Sets the port to listen on for Blob Storage.")

	flag.Parse()

	blobPath, err := filepath.Abs("data/")
	check(err)
	check(InitBlob(blobPath))

	log.Printf("Blob storage listening on port %d...", config.BlobStorage.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.BlobStorage.Port), nil)
	if err != nil {
		panic(err)
	}
}
