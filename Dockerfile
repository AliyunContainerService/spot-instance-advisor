FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/github.com/AliyunContainerService/spot-instance-advisor

COPY ./ /go/src/github.com/AliyunContainerService/spot-instance-advisor

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o dingtalkbot github.com/AliyunContainerService/spot-instance-advisor/cmd/dingtalkbot

FROM alpine:3.10

WORKDIR /root/

RUN apk add --no-cache ca-certificates

COPY --from=builder /go/src/github.com/AliyunContainerService/spot-instance-advisor/dingtalkbot .

ENTRYPOINT ["./dingtalkbot"]
