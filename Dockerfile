# Версию Go держите не ниже строки `go ...` в go.work
ARG GO_VERSION=1.27
ARG ALPINE_VERSION=3.22

# ============ builder: сборка одного сервиса ============
FROM golang:${GO_VERSION}-alpine AS builder

# "|" = пробовать следующий прокси при ЛЮБОЙ ошибке (в т.ч. таймауте), а не только при 404
ARG GOPROXY=https://goproxy.io|https://proxy.golang.org
# sum.golang.org из РФ часто недоступен -> проверка отключена.
# Целостность зависимостей сервисов всё равно проверяется по go.sum.
ARG GOSUMDB=off
# direct и git не используем: прямые походы на GitHub и apk из РФ нестабильны
ENV GOPROXY=${GOPROXY} GOSUMDB=${GOSUMDB} CGO_ENABLED=0 GOFLAGS=-buildvcs=false

WORKDIR /app
COPY go.work go.work.sum ./
COPY pkg ./pkg
COPY contracts ./contracts
COPY services ./services

# Скачивание модулей ДО объявления SERVICE_NAME: шаг одинаков для всех сервисов,
# BuildKit выполнит его один раз и переиспользует. Повторы - из-за нестабильной сети.
RUN --mount=type=cache,target=/go/pkg/mod \
    for i in 1 2 3 4 5 6; do \
      go mod download && exit 0; \
      echo "go mod download: попытка $i не удалась, повтор через 5с"; sleep 5; \
    done; exit 1

ARG SERVICE_NAME
# cache mount: модули и билд-кэш переиспользуются между сервисами и пересборками
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -tags timetzdata \
      -o /out/app ./services/${SERVICE_NAME}/cmd/

# ============ goose-build: сборка goose ============
FROM golang:${GO_VERSION}-alpine AS goose-build
ARG GOPROXY=https://goproxy.io|https://proxy.golang.org
ARG GOSUMDB=off
ENV GOPROXY=${GOPROXY} GOSUMDB=${GOSUMDB} CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    go install github.com/pressly/goose/v3/cmd/goose@v3.26.0

# ============ migrator_final: образ для миграций ============
FROM alpine:${ALPINE_VERSION} AS migrator_final
COPY --from=goose-build /go/bin/goose /usr/local/bin/goose
COPY services /services
CMD ["goose", "-version"]

# ============ service: финальный образ сервиса ============
# apk не используем (меньше походов в сеть): сертификаты берём из builder,
# таймзоны вшиты в бинарник тегом timetzdata
FROM alpine:${ALPINE_VERSION} AS service
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
COPY --from=builder /out/app ./main
# ключи монтируются томом из compose (./secrets -> /app/secrets), в образ не копируются
CMD ["./main"]