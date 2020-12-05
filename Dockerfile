ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest


ARG ARCH="amd64"
ARG OS="linux"
COPY ./prometheus-lightsail-sd	/bin/prometheus-lightsail-sd

USER nobody
EXPOSE 8383
VOLUME [ "/prometheus-lightsail-sd" ]

ENTRYPOINT [ "/bin/prometheus-lightsail-sd" ]
