FROM alpine:3.18

###############################################################################
#                                INSTALLATION
###############################################################################

ENV TZ Asia/Shanghai
RUN apk add alpine-conf tzdata && \
    /sbin/setup-timezone -z Asia/Shanghai && \
    apk del alpine-conf

ENV WORKDIR                                           /app
ADD ./temp/linux_amd64/hitokoto_notification_worker   $WORKDIR/hitokoto_notification_worker
RUN chmod +x $WORKDIR/hitokoto_notification_worker

###############################################################################
#                                   START
###############################################################################
WORKDIR $WORKDIR
CMD     ["./hitokoto_notification_worker"]
