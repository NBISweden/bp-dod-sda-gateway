FROM golang:1.26-alpine AS builder

ENV GOPATH=$PWD
ENV CGO_ENABLED=0

COPY . .

RUN go build -buildvcs=false -o ./bp-dod-sda-gateway
RUN echo "nobody:x:65534:65534:nobody:/:/sbin/nologin" > passwd

FROM gcr.io/distroless/static-debian13

COPY --from=builder /go/bp-dod-sda-gateway /usr/bin/

USER 65534
CMD ["bp-dod-sda-gateway"]