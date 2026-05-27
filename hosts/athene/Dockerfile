FROM alpine:latest
RUN apk add --no-cache soju
COPY soju.conf /etc/soju.conf
COPY soju.db /var/lib/soju/soju.db
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
