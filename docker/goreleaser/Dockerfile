FROM alpine:3.19.0 as builder
RUN apk --update add ca-certificates

FROM scratch
LABEL maintainer "soulteary <soulteary@gmail.com>"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY webhook /usr/bin/webhook
EXPOSE 9000/tcp
CMD ["/usr/bin/webhook"]
