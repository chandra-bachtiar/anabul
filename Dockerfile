# 1) Frontend: build SvelteKit menjadi file statis
FROM node:24-alpine AS web
WORKDIR /web
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci
COPY apps/web/ ./
RUN npm run build

# 2) Backend: embed hasil build frontend lalu kompilasi binary
FROM golang:1.26-alpine AS api
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
COPY --from=web /web/build ./internal/webui/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/anabul ./cmd/server

# 3) Runtime: distroless, hanya binary (migrasi goose di-embed)
FROM gcr.io/distroless/static:nonroot
COPY --from=api /out/anabul /anabul
ENV PORT=8080 \
    TZ=Asia/Jakarta
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/anabul"]
