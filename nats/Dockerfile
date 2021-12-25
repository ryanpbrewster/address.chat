FROM nats:2.6-alpine
COPY nats.conf /etc/nats.conf
ENTRYPOINT ["nats-server", "-c", "/etc/nats.conf"]
