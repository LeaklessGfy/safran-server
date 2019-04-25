FROM golang:1.8

WORKDIR /go/src/app

VOLUME ["/go/src/app"]

#RUN go get -d -v ./...
#RUN go install -v ./...

EXPOSE 8088

CMD ["make", "dockerrun"]
