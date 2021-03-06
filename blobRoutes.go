package main

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

// DELETEs the blob
func blobDelete(w http.ResponseWriter, r *http.Request, ps httprouter.Params, c BlobRequestContext) {
	container := ps.ByName("container")
	blob := ps.ByName("blob")
	blobPath := filepath.Join(blobBase, container, blob)
	if filepath.Clean(blobPath) != blobPath {
		log.Printf("Bad path %s", blobPath)
		http.Error(w, "Bad Request", 400)
	}

	err := os.Remove(blobPath)
	if err != nil && os.IsNotExist(err) {
		http.Error(w, "File Not Found", 404)
		return
	} else if err != nil {
		panic(err)
	}

	http.Error(w, "Accepted", 202)
}

// GETs the blob
func blobGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params, c BlobRequestContext) {
	container := ps.ByName("container")
	blob := ps.ByName("blob")
	blobPath := filepath.Join(blobBase, container, blob)
	if filepath.Clean(blobPath) != blobPath {
		log.Printf("Bad path %s", blobPath)
		http.Error(w, "Bad Request", 400)
	}

	blobData, err := ioutil.ReadFile(blobPath)
	if err != nil && os.IsNotExist(err) {
		http.Error(w, "File Not Found", 404)
		return
	} else if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	if r.Form.Get("rscd") != "" {
		w.Header().Set("Content-Disposition", r.Form.Get("rscd"))
	}

	w.Write(blobData)
}

// Handles a PUT on a blob (creates the blob, block, or commits the block list)
func blobPut(w http.ResponseWriter, r *http.Request, ps httprouter.Params, c BlobRequestContext) {
	container := ps.ByName("container")
	blob := ps.ByName("blob")
	blobPath := filepath.Join(blobBase, container, blob)
	blobUncommittedPath := filepath.Join(blobBase, container, "uncommitted", blob)
	if filepath.Clean(blobPath) != blobPath {
		log.Printf("Bad path %s", blobPath)
		http.Error(w, "Bad Request", 400)
	}
	if filepath.Clean(blobUncommittedPath) != blobUncommittedPath {
		log.Printf("Bad path %s", blobUncommittedPath)
		http.Error(w, "Bad Request", 400)
	}

	blobData, err := ioutil.ReadAll(r.Body)
	check(err)
	defer r.Body.Close()

	comp := r.FormValue("comp")
	if comp == "block" {
		blockid := r.FormValue("blockid")
		blockPath := blobUncommittedPath + "_" + blockid

		if filepath.Clean(blockPath) != blockPath {
			log.Printf("Bad path %s", blockPath)
			http.Error(w, "Bad Request", 400)
			return
		}

		err = ioutil.WriteFile(blockPath, blobData, 0777)
		check(err)

		http.Error(w, "Created", 201)
	} else if comp == "blocklist" {
		list := BlobBlockListResponse{}
		err := xml.Unmarshal(blobData, &list)
		if err != nil {
			log.Println("Bad block list")
			http.Error(w, "Bad Request", 400)
			return
		}

		file, err := os.Create(blobPath)
		check(err)
		defer file.Close()
		for _, blockId := range list.Blocks {
			blockPath := blobUncommittedPath + "_" + blockId
			blockData, err := ioutil.ReadFile(blockPath)
			check(err)
			os.Remove(blockPath)
			_, err = file.Write(blockData)
			check(err)
		}

		http.Error(w, "Created", 201)
	} else {
		err = ioutil.WriteFile(blobPath, blobData, 0777)
		check(err)

		http.Error(w, "Created", 201)
	}
}

// Handles a GET on a container (lists blobs)
func blobContainerGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params, c BlobRequestContext) {
	container := ps.ByName("container")
	prefix := r.FormValue("prefix")

	containerPath := filepath.Join(blobBase, container)
	if filepath.Clean(containerPath) != containerPath {
		log.Printf("Bad path %s", containerPath)
		http.Error(w, "Bad Request", 400)
		return
	}

	blobs := []BlobResult{}
	containerInfo, err := ioutil.ReadDir(containerPath)
	check(err)

	for _, file := range containerInfo {
		if !file.IsDir() {
			if prefix != "" && !strings.HasPrefix(file.Name(), prefix) {
				continue
			}
			blobs = append(blobs, BlobResult{
				Name:     file.Name(),
				Snapshot: time.Now(),
				Properties: BlobPropertiesResult{
					LastModified:          TimeRFC1123(file.ModTime()),
					Etag:                  "",
					ContentMD5:            "",
					ContentLength:         file.Size(),
					ContentType:           "",
					ContentEncoding:       "",
					CacheControl:          "",
					ContentLanguage:       "",
					ContentDisposition:    "",
					BlobType:              BlobTypeBlock,
					SequenceNumber:        0,
					CopyID:                "",
					CopyStatus:            "",
					CopySource:            "",
					CopyProgress:          "",
					CopyCompletionTime:    TimeRFC1123(time.Now()),
					CopyStatusDescription: "",
					LeaseStatus:           "",
					LeaseState:            "",
					LeaseDuration:         "",
					ServerEncrypted:       false,
					IncrementalCopy:       false,
				},
				Metadata: "",
			})
		}
	}

	err = writeXML(w, BlobEnumerationResults{
		xml.Name{},
		"",
		"",
		0,
		"",
		blobs,
		"",
	})
	check(err)
}

// Handles a PUT on a container (creates the container)
func blobContainerPut(w http.ResponseWriter, r *http.Request, ps httprouter.Params, c BlobRequestContext) {
	restype := r.FormValue("restype")
	container := ps.ByName("container")

	if restype == "container" {
		containerPath := filepath.Join(blobBase, container)
		if filepath.Clean(containerPath) != containerPath {
			log.Printf("Bad path %s", containerPath)
			http.Error(w, "Bad Request", 400)
			return
		}
		containerUncommittedPath := filepath.Join(containerPath, "uncommitted")
		_, err := os.Stat(containerPath)
		if err != nil {
			err = os.Mkdir(containerPath, 0777)
			check(err)
			err = os.Mkdir(containerUncommittedPath, 0777)
			check(err)
			http.Error(w, "Created", 201)
		} else {
			http.Error(w, "The specified container already exists.", 409)
			writeXML(w, BlobErrorResponse{
				Code:    "ContainerAlreadyExists",
				Message: "The specified container already exists.",
			})
		}
	} else {
		http.Error(w, "Bad Request", 400)
	}
}
