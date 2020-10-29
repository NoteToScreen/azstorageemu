FROM golang:1.15.3-alpine

WORKDIR /go/src/github.com/NoteToScreen/azstorageemu
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 10000
VOLUME ["/go/src/github.com/NoteToScreen/azstorageemu/data"]

CMD ["azstorageemu"]
