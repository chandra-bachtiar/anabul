# 1) Frontend: build SvelteKit into static files
FROM node:25-alpine AS web
WORKDIR /web
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci
COPY apps/web/ ./
RUN npm run build

# 2) Backend: embed the frontend build, then compile the binary
FROM golang:1.27-alpine AS api
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
COPY --from=web /web/build ./internal/webui/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/anabul ./cmd/server

# 3) Runtime: distroless image with just the binary (goose migrations are embedded)
FROM gcr.io/distroless/static:nonroot
COPY --from=api /out/anabul /anabul
ENV PORT=8080 \
    TZ=Asia/Jakarta
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/anabul"]
