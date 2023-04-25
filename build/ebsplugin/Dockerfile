FROM golang:1.18.7 as builder
COPY . /go/src/github.com/volcengine/volcengine-csi-driver
WORKDIR /go/src/github.com/volcengine/volcengine-csi-driver
RUN make build-linux CMDS=ebsplugin

FROM debian:buster
COPY --from=builder /go/src/github.com/volcengine/volcengine-csi-driver/bin/ebsplugin /ebsplugin
RUN apt-get update &&  \
    apt-get -y install ca-certificates mount && \
    apt-get clean all && \
    chmod +x /ebsplugin
ENTRYPOINT ["/ebsplugin"]