FROM golang:1.16.2-alpine3.13
ADD go.mod /app/
WORKDIR /app
RUN go mod download
ADD auth-client.go /app/
RUN go build .


FROM alpine:3.13.2

RUN  apk --no-cache add --update iptables openssl openvpn
RUN  rm -rf /var/cache/apk/*

COPY    /entrypoint.sh /
COPY --from=0 /app/auth-client /openvpn/auth-client

ENTRYPOINT  ["/entrypoint.sh"]
