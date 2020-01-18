FROM golang:1.13 AS build

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -a -ldflags '-extldflags "-static"'

FROM alpine:latest AS certs
RUN apk --no-cache add ca-certificates

FROM scratch
ENV PORT 4080
EXPOSE 4080
COPY --from=build /app/wobbotfet .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["./wobbotfet"]
