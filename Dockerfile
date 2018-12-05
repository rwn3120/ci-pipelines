FROM golang:alpine as builder
RUN mkdir /tmp/ci-pipelines
COPY go.mod *.go gitlab /tmp/ci-pipelines/
RUN go build -o /usr/bin/ci-pipelines /tmp/ci-pipelines

FROM alpine:latest  
COPY --from=builder /usr/bin/ci-pipelines /usr/bin/ci-pipelines
EXPOSE 1111
CMD ["/usr/bin/ci-pipelines"]  
