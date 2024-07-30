# build stage
FROM golang:1.22-alpine AS build

ADD . /go/build
WORKDIR /go/build

ADD go.mod go.sum /go/build/
RUN go mod download

# install gcc
RUN apk add build-base

RUN go build -o vote main.go

# final stage
FROM alpine:3.18

# install timezone data
RUN apk add --no-cache tzdata
ENV TZ Asia/Taipei
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

COPY --from=build /go/build/vote /var/application/vote
COPY --from=build /go/build/config /var/application/config

EXPOSE 8080

WORKDIR /var/application
CMD [ "./vote" ]