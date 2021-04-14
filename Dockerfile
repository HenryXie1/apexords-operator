# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace
# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# add proxy for building behind firewall
#ENV http_proxy=http://<proxy-server>:80
#ENV https_proxy=http://<proxy-server>:80


# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /workspace/manager .
ENTRYPOINT ["/manager"]
