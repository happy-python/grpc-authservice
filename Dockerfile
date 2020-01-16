FROM alpine:3.10

LABEL maintainer="jack <yongjie.zhang@henganpros.com>"

RUN apk --no-cache add ca-certificates

ENV ADDRESS :20020

COPY main /usr/local/bin/

CMD ["main"]

EXPOSE 20020