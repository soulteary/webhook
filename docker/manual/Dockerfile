FROM golang:1.22-bullseye AS Builder
ENV GOPROXY=https://goproxy.cn
ENV CGO_ENABLED=0
WORKDIR /app
COPY . .
RUN go mod download -x
RUN go build -ldflags "-w -s" -o webhook .

FROM debian:stretch
LABEL maintainer "soulteary <soulteary@gmail.com>"
COPY --from=builder /app/webhook /bin/
EXPOSE 9000/tcp
CMD webhook
