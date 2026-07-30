FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/lyra ./cmd/lyra

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg ca-certificates && rm -rf /var/lib/apt/lists/* && useradd --system --uid 10001 lyra
COPY --from=build /out/lyra /usr/local/bin/lyra
USER lyra
EXPOSE 8080
ENTRYPOINT ["lyra"]
CMD ["serve"]
