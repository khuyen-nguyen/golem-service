FROM python:alpine3.17

WORKDIR /app

RUN apk update && apk upgrade \
  && apk add --no-cache --update build-base cmake git curl bash openssh-server util-linux aws-cli

ENTRYPOINT ["/entrypoint/golem-service-entrypoint-linux-amd64"]

COPY . /app/
