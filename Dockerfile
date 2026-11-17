FROM node:24-alpine AS frontend
WORKDIR /src
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /src
COPY backend/go.* ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /livepoll ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -H app
COPY --from=backend /livepoll /usr/local/bin/livepoll
COPY --from=frontend /src/dist /app/public
ENV STATIC_DIR=/app/public GIN_MODE=release
EXPOSE 8080
USER app
ENTRYPOINT ["/usr/local/bin/livepoll"]
