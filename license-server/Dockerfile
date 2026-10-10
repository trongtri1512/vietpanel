# Build: docker build -t vietpanel/license-server .
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
COPY templates/ templates/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/license-server .

FROM alpine:3.20
RUN adduser -D -u 10001 license && mkdir -p /data && chown license:license /data
COPY --from=build /out/license-server /usr/local/bin/license-server
USER license
ENV LICENSE_DB=/data/licenses.json \
    LICENSE_ADDR=:2095
VOLUME ["/data"]
EXPOSE 2095
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:2095/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/license-server"]
