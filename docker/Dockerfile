# build stage
FROM golang:alpine AS build-env

ARG version="0.0.0"
ARG prerelease""
ARG githash=""
ARG buildstamp=""

RUN apk add --no-cache git openssh-client gcc musl-dev

WORKDIR /go
ENV GOPATH=/go
ENV GO111MODULE=on
RUN go version
ADD . /go/src/github.com/YaleSpinup/reaper/
RUN cd /go/src/github.com/YaleSpinup/reaper && go get ./...
RUN cd /go/src/github.com/YaleSpinup/reaper && go build -o /go/reaper -ldflags "-X main.Version=$version -X main.VersionPrerelease=$prerelease -X main.githash=$githash -X main.buildstamp=$buildstamp" *.go
RUN  ./reaper -version

# final stage
FROM alpine
LABEL maintainer="Camden Fisher <camden.fisher@yale.edu>"

WORKDIR /app
COPY --from=build-env /go/reaper /app/reaper
RUN chmod 555 /app/reaper

RUN apk add --no-cache bash ca-certificates wget python gettext tzdata && \
    wget -nv "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" && \
    unzip awscli-bundle.zip && \
    ./awscli-bundle/install -i /usr/local/aws -b /usr/local/bin/aws && \
    /usr/local/bin/aws --version

# Install Deco
ARG DECO_VERSION=0.2.2
ARG DECO_OS=linux
ARG DECO_ARCH=amd64
ADD https://github.com/YaleUniversity/deco/releases/download/v${DECO_VERSION}/deco-v${DECO_VERSION}-${DECO_OS}-${DECO_ARCH} /usr/local/bin/deco
RUN chmod 555 /usr/local/bin/deco && deco version

COPY docker/*_config.sh /app
COPY docker/config.deco.json /app/config/config.json
RUN chmod 600 /app/config/config.json

RUN chown -R nobody:nogroup /app
USER nobody

CMD /app/import_config.sh && /app/reaper -config /app/config/config.json