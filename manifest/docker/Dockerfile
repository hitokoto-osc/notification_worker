FROM alpine:3.18

###############################################################################
#                                INSTALLATION
###############################################################################

ENV TZ Asia/Shanghai
RUN apk add alpine-conf tzdata && \
    /sbin/setup-timezone -z Asia/Shanghai && \
    apk del alpine-conf

ENV WORKDIR                                           /app
VOLUME                                                $WORKDIR/data
ADD ./config.example.yml                              $WORKDIR/config.yml
ADD ./notification-worker                             $WORKDIR/notification-worker
RUN chmod +x $WORKDIR/notification-worker

###############################################################################
#                                   START
###############################################################################
WORKDIR $WORKDIR
CMD     ["./notification-worker"]
