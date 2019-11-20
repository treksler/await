FROM golang:1.13.4-alpine3.10 AS binary
RUN apk -U add openssl git upx

ADD . /go/src/github.com/treksler/await
WORKDIR /go/src/github.com/treksler/await

RUN go get github.com/robfig/glock
RUN glock sync -n < GLOCKFILE
RUN CGO_ENABLED=0 go install -ldflags="-s -w"
RUN upx --brute /go/bin/await

FROM scratch
MAINTAINER Risto Treksler <risto78@gmail.com>

COPY --from=binary /go/bin/await /usr/bin/await

ENTRYPOINT ["await"]
CMD ["--help"]
