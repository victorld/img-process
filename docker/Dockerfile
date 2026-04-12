FROM node:20-alpine AS web-builder

WORKDIR /app/web
COPY web/package.json web/tsconfig.json web/tsconfig.app.json web/vite.config.ts web/index.html ./
COPY web/src ./src
RUN npm install
RUN npm run build

FROM golang:1.20.11-alpine3.18 AS go-builder

WORKDIR /app
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY api ./api
COPY bootstrap ./bootstrap
COPY cons ./cons
COPY dao ./dao
COPY main ./main
COPY middleware ./middleware
COPY model ./model
COPY plugin ./plugin
COPY route ./route
COPY rpc ./rpc
COPY service ./service
COPY tools ./tools
COPY config.yaml ./config.yaml
RUN CGO_ENABLED=0 go build -mod=vendor -o /bin/img-process ./main/webserver/webserver_main.go

FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata perl-image-exiftool

COPY --from=go-builder /bin/img-process /app/img-process
COPY --from=web-builder /app/web/dist /app/web/dist
COPY config.yaml /app/config.yaml
COPY docker/entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/entrypoint.sh && mkdir -p /app/log

EXPOSE 8081

ENTRYPOINT ["/app/entrypoint.sh"]
