FROM alpine:3.24.1 AS certificates
RUN apk add --no-cache ca-certificates && \
    mkdir -p /var/lib/webhook/hooks

FROM scratch
LABEL maintainer "soulteary <soulteary@gmail.com>"
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=certificates --chown=65532:65532 /var/lib/webhook /var/lib/webhook
COPY --chown=65532:65532 webhook /usr/bin/webhook
USER 65532:65532
WORKDIR /var/lib/webhook
EXPOSE 9000/tcp
ENTRYPOINT ["/usr/bin/webhook"]
