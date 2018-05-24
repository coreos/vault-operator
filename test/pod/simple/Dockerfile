# golang:X-alpine can't be used since it does not support the race detector flag which assumes a glibc based system, whereas alpine linux uses musl libc
# https://github.com/golang/go/issues/14481
FROM golang:1.10.2 as builder

ADD ./ /go/src/github.com/coreos/vault-operator

WORKDIR /go/src/github.com/coreos/vault-operator

RUN go test ./test/e2e/ -c -o /bin/vault-operator-e2e --race

FROM busybox:1.28.3-glibc

COPY --from=builder /bin/vault-operator-e2e /bin
COPY --from=builder /go/src/github.com/coreos/vault-operator/test/pod/simple/run-e2e /bin/run-e2e

CMD ["/bin/run-e2e"]
