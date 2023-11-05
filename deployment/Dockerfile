FROM golang:1.21.3 AS builder

ENV GO111MODULE=on
ENV GOFLAGS=-mod=vendor
ENV APP_HOME /go/src/app

RUN mkdir -p "$APP_HOME"
WORKDIR "$APP_HOME"

ADD .. .

RUN go mod tidy
RUN go mod vendor
RUN go build -o main cmd/main.go


FROM scratch

WORKDIR /app
COPY --from=builder /go/src/app/main .

CMD [ "./main" ]