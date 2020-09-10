ARG BASE="alpine:3.12.0"
ARG ARCH="amd64"
ARG OS="linux"
FROM --platform=${OS}/${ARCH} ${BASE}

ARG ARCH
ARG OS
COPY .build/${OS}-${ARCH}/prometheus-lightsail-sd /bin/prometheus-lightsail-sd

RUN mkdir /var/prometheus-lightsail-sd && \
    chown nobody:nobody /var/prometheus-lightsail-sd

EXPOSE      8383
USER        nobody
ENTRYPOINT  /bin/prometheus-lightsail-sd --output.file="/var/prometheus-lightsail-sd/lightsail_sd.json"
