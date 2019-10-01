# build stage
FROM golang:alpine AS build-env
RUN apk add --no-cache git openssh-client gcc musl-dev
RUN mkdir /app
WORKDIR /app
RUN go version
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o /go/api *.go

# final stage
FROM alpine
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build-env /go/api /app/api
EXPOSE 80
ENTRYPOINT ["./api"]
