# Minimal, dependency-free build. Produces a static binary on distroless.
# Optional: the binary runs standalone without a container.

FROM golang:1.22 AS build
WORKDIR /src
# Module first for layer caching; there are no external dependencies to fetch.
COPY go.mod ./
COPY . .
# CGO off => fully static binary; assets and fonts are embedded via embed.FS.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /monero-server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /monero-server /monero-server
EXPOSE 8080
ENV ADDR=:8080
USER nonroot:nonroot
ENTRYPOINT ["/monero-server"]
