# Stage 1: The builder
FROM golang:1.24.7 AS builder

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /main .

# Stage 2: The final production image
FROM alpine:3.18

# Set the working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /main .

# Expose the service port
EXPOSE 8080

# Command to run the executable
CMD ["/app/main"]