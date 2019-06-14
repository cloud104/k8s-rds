# Build the manager binary
FROM golang:alpine as builder

RUN apk add --no-cache -u git
WORKDIR /workspace
# Copy the go source
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY cmd/ cmd/
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o entrypoint cmd/*.go

# Use distroless as minimal base image to package the entrypoint binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /workspace/entrypoint .
ENTRYPOINT ["/entrypoint"]
CMD ["server", "--provider", "aws"]
