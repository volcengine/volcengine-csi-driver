FROM golang:1.18.7 as builder
COPY . /go/src/github.com/volcengine/volcengine-csi-driver
WORKDIR /go/src/github.com/volcengine/volcengine-csi-driver
RUN make build-linux CMDS=nasplugin

FROM debian:buster
COPY --from=builder /go/src/github.com/volcengine/volcengine-csi-driver/bin/nasplugin /nasplugin
RUN apt-get update &&  \
    apt-get -y install ca-certificates mount nfs-common nfs-kernel-server && \
    apt-get clean all && \
    chmod +x /nasplugin

ENTRYPOINT ["/nasplugin", "run"]