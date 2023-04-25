FROM golang:1.18.7 as builder
COPY . /go/src/github.com/volcengine/volcengine-csi-driver
WORKDIR /go/src/github.com/volcengine/volcengine-csi-driver
RUN make build-linux CMDS=tosplugin

FROM debian:buster
COPY --from=builder /go/src/github.com/volcengine/volcengine-csi-driver/bin/tosplugin /tosplugin
ENTRYPOINT ["/tosplugin", "run"]