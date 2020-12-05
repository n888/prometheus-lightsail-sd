FROM alpine:3.12

ARG TARGETOS="linux"
ARG TARGETARCH="amd64"

COPY ./prometheus-lightsail-sd /bin/prometheus-lightsail-sd

RUN adduser -u 888 -D prometheus       && \
    mkdir /home/prometheus/.aws        && \
    mkdir /var/prometheus-lightsail-sd && \
    chown prometheus:prometheus /home/prometheus/.aws /var/prometheus-lightsail-sd

EXPOSE     9888
USER       prometheus
VOLUME     ["/home/prometheus/.aws"]

ENTRYPOINT ["/bin/prometheus-lightsail-sd"]
CMD        ["--output.file=/var/prometheus-lightsail-sd/lightsail_sd.json"]
