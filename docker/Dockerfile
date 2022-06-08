# build stage
FROM golang:alpine AS build-env

ARG version="0.0.0"
ARG prerelease=""
ARG githash=""
ARG buildstamp=""

RUN apk add --no-cache git openssh-client gcc musl-dev

RUN mkdir /app
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go test ./... -cover
RUN go build -o /app/api -ldflags="-X main.Version=$version -X main.VersionPrerelease=$prerelease -X main.githash=$githash -X main.buildstamp=$buildstamp" *.go
RUN ./api -version

# final stage
FROM alpine

WORKDIR /app
COPY --from=build-env /app/api /app/api
RUN chmod 555 /app/api

RUN apk add --no-cache bash ca-certificates tzdata

# Install Deco
ARG DECO_VERSION=1.3.0
ARG DECO_OS=linux
ARG DECO_ARCH=amd64
ADD https://github.com/YaleUniversity/deco/releases/download/v${DECO_VERSION}/deco_${DECO_VERSION}_${DECO_OS}_${DECO_ARCH}.tar.gz /usr/local/bin/deco.tar.gz
RUN cd /usr/local/bin && tar -zxvf deco.tar.gz && rm -f deco.tar.gz && chmod 555 deco && /usr/local/bin/deco version

COPY docker/*_config.sh /app
COPY docker/config.deco.json /app/config/config.json
RUN chmod 600 /app/config/config.json

RUN chown -R nobody:nogroup /app
USER nobody

CMD /app/import_config.sh && /app/api -config /app/config/config.json