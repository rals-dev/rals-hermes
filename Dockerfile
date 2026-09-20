# syntax=docker/dockerfile:1.7
# Multi-stage build (T-401): Node builds the Vue app, Go embeds it into a
# static binary, and the runtime image is distroless with a non-root user.

FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION}" \
      -o /out/bff ./cmd/bff

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/bff /app/bff
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/bff"]
CMD ["-config", "/app/config.yaml"]
