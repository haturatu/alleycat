FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /pb/pocketbase ./cmd/pocketbase

FROM alpine:3.24
WORKDIR /pb
RUN apk add --no-cache libwebp-tools
COPY --from=build /pb/pocketbase /pb/pocketbase
EXPOSE 8091
VOLUME ["/pb/pb_data"]
CMD ["/pb/pocketbase", "serve", "--http=0.0.0.0:8091"]
