FROM alpine:3 AS certs
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -g '' -s /sbin/nologin appuser

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=certs /etc/passwd /etc/passwd
COPY bitcoind-exporter /usr/bin/bitcoind-exporter

USER appuser
EXPOSE 3000
ENTRYPOINT ["/usr/bin/bitcoind-exporter"]
