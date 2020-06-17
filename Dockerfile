FROM golang:1.14 as builder

WORKDIR /

ADD main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -o ipvs-logger main.go

FROM scratch

COPY --from=builder /ipvs-logger /ipvs-logger

ENTRYPOINT [ "/ipvs-logger" ]