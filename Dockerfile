FROM golang:alpine as builder
RUN mkdir -p "/build"
ADD . "/build/"
WORKDIR "/build"
RUN CGO_ENABLED=0 GOOS=linux go build -o "/usr/bin/ci-pipelines" 

FROM alpine:latest  
ADD web "/web"
COPY --from=builder "/usr/bin/ci-pipelines" "/usr/bin/ci-pipelines"
EXPOSE 1111
CMD ["/usr/bin/ci-pipelines"]  
