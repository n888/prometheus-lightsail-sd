ARG BASE="alpine:3.12"
FROM ${BASE}

ARG TARGETOS
ARG TARGETARCH

USER root

COPY .build/${TARGETOS}-${TARGETARCH}/prometheus-lightsail-sd /bin/prometheus-lightsail-sd

RUN adduser -u 888 -D prometheus && \
    mkdir /home/prometheus/.aws && \
    mkdir /prometheus-lightsail-sd && \
    chown nobody:nogroup /prometheus-lightsail-sd

USER       nobody
EXPOSE     9888
VOLUME     ["/prometheus-lightsail-sd"]
WORKDIR    /prometheus-lightsail-sd
ENTRYPOINT ["/bin/prometheus-lightsail-sd"]
CMD        ["--output.file=/prometheus-lightsail-sd/lightsail_sd.json"]
