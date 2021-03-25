FROM golang:1.16.2-alpine3.13
ADD go.mod /app/
WORKDIR /app
RUN go mod download
ADD auth-server.go /app/
RUN go build .


FROM alpine:3.13.2

COPY --from=0 /app/auth-server /app/auth-server
ADD /client.ovpn.tpl /app/client.ovpn.tpl

ENTRYPOINT  ["/app/auth-server"]
