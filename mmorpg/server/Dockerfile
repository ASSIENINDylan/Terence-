# Build du serveur MMORPG — binaire statique Go (image finale minuscule, §12).
FROM golang:1.24-alpine AS build
WORKDIR /src

# Cache des dépendances.
COPY go.mod go.sum* ./
RUN go mod download

# Code source + build statique.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# Image d'exécution minimale.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
