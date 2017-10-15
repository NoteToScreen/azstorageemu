package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

var blobAccountKey, _ = base64.StdEncoding.DecodeString("Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==")

const blobAccountName = "devstoreaccount1"

var blobBase string

type blobAPIRouteHandle func(http.ResponseWriter, *http.Request, httprouter.Params, BlobRequestContext)

type BlobResult struct {
	Name       string
	Snapshot   time.Time
	Properties BlobPropertiesResult
	Metadata   string
}

type BlobPropertiesResult struct {
	LastModified          TimeRFC1123 `xml:"Last-Modified"`
	Etag                  string      `xml:"Etag"`
	ContentMD5            string      `xml:"Content-MD5" header:"x-ms-blob-content-md5"`
	ContentLength         int64       `xml:"Content-Length"`
	ContentType           string      `xml:"Content-Type" header:"x-ms-blob-content-type"`
	ContentEncoding       string      `xml:"Content-Encoding" header:"x-ms-blob-content-encoding"`
	CacheControl          string      `xml:"Cache-Control" header:"x-ms-blob-cache-control"`
	ContentLanguage       string      `xml:"Cache-Language" header:"x-ms-blob-content-language"`
	ContentDisposition    string      `xml:"Content-Disposition" header:"x-ms-blob-content-disposition"`
	BlobType              BlobType    `xml:"BlobType"`
	SequenceNumber        int64       `xml:"x-ms-blob-sequence-number"`
	CopyID                string      `xml:"CopyId"`
	CopyStatus            string      `xml:"CopyStatus"`
	CopySource            string      `xml:"CopySource"`
	CopyProgress          string      `xml:"CopyProgress"`
	CopyCompletionTime    TimeRFC1123 `xml:"CopyCompletionTime"`
	CopyStatusDescription string      `xml:"CopyStatusDescription"`
	LeaseStatus           string      `xml:"LeaseStatus"`
	LeaseState            string      `xml:"LeaseState"`
	LeaseDuration         string      `xml:"LeaseDuration"`
	ServerEncrypted       bool        `xml:"ServerEncrypted"`
	IncrementalCopy       bool        `xml:"IncrementalCopy"`
}

// BlobType defines the type of the Azure Blob.
type BlobType string

// Types of page blobs
const (
	BlobTypeBlock  BlobType = "BlockBlob"
	BlobTypePage   BlobType = "PageBlob"
	BlobTypeAppend BlobType = "AppendBlob"
)

type BlobRequestContext struct {
}

func blobAPIRoute(h blobAPIRouteHandle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r.ParseForm()

		// check authorization
		if r.Form.Get("sig") != "" {
			// it's using a shared access signature
			stringToSign := r.Form.Get("sp") + "\n" +
				r.Form.Get("st") + "\n" +
				r.Form.Get("se") + "\n" +
				fmt.Sprintf("/blob%s", r.URL.Path) + "\n" +
				r.Form.Get("si") + "\n" +
				r.Form.Get("sip") + "\n" +
				r.Form.Get("spr") + "\n" +
				r.Form.Get("sv") + "\n" +
				r.Form.Get("rscc") + "\n" +
				r.Form.Get("rscd") + "\n" +
				r.Form.Get("rsce") + "\n" +
				r.Form.Get("rscl") + "\n" +
				r.Form.Get("rsct")

			// RFC3339 == ISO 8601
			signedExpiry, err := time.Parse(time.RFC3339, r.Form.Get("se"))
			if err != nil {
				log.Println("SAS signed expiry wasn't valid")
				http.Error(w, "Bad Request - Invalid SAS signed expiry", 400)
				return
			}

			if signedExpiry.Before(time.Now()) {
				log.Println("SAS signature expired")
				http.Error(w, "Bad Request - SAS signature expired", 400)
				return
			}

			mac := hmac.New(sha256.New, blobAccountKey)
			mac.Write([]byte(stringToSign))
			expected := mac.Sum(nil)

			inputSig, err := base64.StdEncoding.DecodeString(r.Form.Get("sig"))

			if err != nil || !hmac.Equal(inputSig, expected) {
				log.Println("SAS signature wasn't valid")
				http.Error(w, "Bad Request", 400)
				return
			}
		} else {
			// it's using an authorization header
			// see https://docs.microsoft.com/en-us/rest/api/storageservices/authentication-for-the-azure-storage-services
			auth := r.Header.Get("Authorization")
			authParts := strings.Split(auth, " ")
			if authParts[0] != "SharedKey" {
				log.Println("Authorization header wasn't using SharedKey")
				http.Error(w, "Bad Request", 400)
				return
			}

			keyParts := strings.Split(authParts[1], ":")
			inputKey, err := base64.StdEncoding.DecodeString(keyParts[1])
			if err != nil {
				log.Println("SharedKey wasn't valid base64")
				http.Error(w, "Bad Request", 400)
				return
			}

			// create string to sign
			stringToSign := ""
			headers := []string{
				"Content-Encoding",
				"Content-Language",
				"Content-Length",
				"Content-MD5",
				"Content-Type",
				"Date",
				"If-Modified-Since",
				"If-Match",
				"If-None-Match",
				"If-Unmodified-Since",
				"Range",
			}

			// add method and signed headers
			stringToSign += r.Method + "\n"
			for _, headerName := range headers {
				if headerName == "Content-Length" && r.Header.Get("Content-Length") == "0" {
					// special exception - treat it as blank
					stringToSign += "\n"
				} else {
					stringToSign += r.Header.Get(headerName) + "\n"
				}
			}

			// create canonicalized header string
			msHeaderNames := []string{}
			for name, _ := range r.Header {
				if strings.HasPrefix(strings.ToLower(name), "x-ms") {
					msHeaderNames = append(msHeaderNames, strings.ToLower(name))
				}
			}
			sort.Strings(msHeaderNames)
			canonicalizedHeaders := ""
			for _, name := range msHeaderNames {
				canonicalizedHeaders += name
				canonicalizedHeaders += ":"
				canonicalizedHeaders += r.Header.Get(name)
				canonicalizedHeaders += "\n"
			}

			canonicalizedResource := "/" + blobAccountName
			canonicalizedResource += r.URL.EscapedPath() + "\n"
			params := []string{}
			lowerToOriginalCase := map[string]string{}
			for name, _ := range r.Form {
				lowerToOriginalCase[strings.ToLower(name)] = name
				params = append(params, strings.ToLower(name))
			}
			sort.Strings(params)
			first := true
			for _, paramName := range params {
				if first {
					first = false
				} else {
					canonicalizedResource += "\n"
				}
				paramValues := r.Form[lowerToOriginalCase[paramName]]
				canonicalizedResource += fmt.Sprintf("%s:%s", paramName, strings.Join(paramValues, ","))
			}

			canonicalizedResource = strings.TrimRight(canonicalizedResource, "\n")

			stringToSign += canonicalizedHeaders
			stringToSign += canonicalizedResource

			mac := hmac.New(sha256.New, blobAccountKey)
			mac.Write([]byte(stringToSign))
			expected := mac.Sum(nil)

			if !hmac.Equal(inputKey, expected) {
				log.Println("SharedKey wasn't valid")
				fmt.Println(hex.Dump([]byte(stringToSign)))
				http.Error(w, "Bad Request", 400)
				return
			}
		}

		h(w, r, ps, BlobRequestContext{})
	}
}

func initBlobRoutes() {
	prefix := "/devstoreaccount1"
	router := httprouter.New()

	router.DELETE(prefix+"/:container/:blob", blobAPIRoute(blobDelete))
	router.GET(prefix+"/:container/:blob", blobAPIRoute(blobGet))
	router.PUT(prefix+"/:container/:blob", blobAPIRoute(blobPut))

	router.GET(prefix+"/:container", blobAPIRoute(blobContainerGet))
	router.PUT(prefix+"/:container", blobAPIRoute(blobContainerPut))

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			// go away
			http.Error(w, "Not Found", 404)
			return
		}
		log.Println("404 not found")
		log.Println("Maybe something not implemented?")
		log.Printf("%s %s", r.Method, r.URL.String())
		http.Error(w, "Not Found", 404)
	})

	http.Handle("/", router)
}

func InitBlob(path string) error {
	initBlobRoutes()

	blobBase = filepath.Join(path, "blob/")
	info, err := os.Stat(blobBase)
	if err != nil || !info.IsDir() {
		err = os.MkdirAll(blobBase, 0777)
		if err != nil {
			return err
		}
	}

	return nil
}
