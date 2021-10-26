FROM alpine:latest
RUN apk --no-cache add bash curl jq
ADD run.sh /run.sh
RUN chmod +x /run.sh
CMD /run.sh