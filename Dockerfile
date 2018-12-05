FROM golang:alpine as builder
RUN mkdir -p "${GOPATH}/src/github.com/rwn3120/ci-pipelines"
COPY go.mod *.go gitlab "${GOPATH}/src/github.com/rwn3120/ci-pipelines/"
RUN go build -o "/usr/bin/ci-pipelines" "${GOPATH}/src/github.com/rwn3120/ci-pipelines/"

FROM alpine:latest  
COPY --from=builder "/usr/bin/ci-pipelines" "/usr/bin/ci-pipelines"
EXPOSE 1111
CMD ["/usr/bin/ci-pipelines"]  
