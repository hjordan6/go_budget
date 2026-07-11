# --- Stage 1: build the Vue SPA ---
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- Stage 2: build the Go binary ---
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server .

# --- Stage 3: minimal runtime ---
FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=web /web/dist /app/web/dist
ENV STATIC_DIR=/app/web/dist
ENV APP_PORT=8891
EXPOSE 8891
CMD ["/app/server"]
