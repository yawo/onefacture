FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/onefacture-api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/onefacture-api /usr/local/bin/onefacture-api
COPY internal/storage/migrations /app/migrations
COPY docs/specs /app/specs
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/onefacture-api"]
