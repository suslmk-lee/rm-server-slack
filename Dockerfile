FROM golang:1.21-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod go.sum *.go ./
COPY common ./common
COPY notification ./notification
COPY storage ./storage
COPY *.properties ./

RUN ls -al

RUN apk add --no-cache ca-certificates

RUN go mod download

RUN go build -o main .

WORKDIR /dist

RUN cp /build/main .
RUN cp /build/*.properties .

FROM scratch

COPY --from=builder /dist/main .
COPY --from=builder /dist/*.properties .

ENTRYPOINT ["/main"]