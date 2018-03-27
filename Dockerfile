FROM alpine:latest
RUN apk update && apk add curl ca-certificates && update-ca-certificates
RUN apk add iputils && rm -rf /var/cache/apk/*
ADD ipalloc /
CMD ["/ipalloc"]
