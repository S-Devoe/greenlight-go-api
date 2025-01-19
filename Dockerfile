# Build Stage
FROM golang:1.23.4-alpine3.20 AS builder

# Set working directory inside the container
WORKDIR /app

# Copy the entire project into the container
COPY . .

# Run `go build` to build the binary using the correct build command
RUN go build -ldflags="-s" -o ./bin/api ./cmd/api

# Run Stage
FROM alpine:3.20

WORKDIR /app

# Copy the built binary from the build stage
COPY --from=builder /app/bin/api .

# Expose the API port
EXPOSE 4000

# Command to run the Go binary when the container starts
CMD ["/app/api"]
