FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/jymw .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata python3 util-linux \
    && adduser -D -u 10001 jiying

WORKDIR /app
COPY --from=build /out/jymw /app/jymw
COPY views2-patch/assets /app/views2/assets
RUN mkdir -p /data && chown -R jiying:jiying /data /app

USER jiying
ENV JIYING_RES=/app \
    JIYING_DATA=/data \
    JIYING_PORT=8080
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/app/jymw"]
