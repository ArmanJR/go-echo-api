# Use the official Golang image as the base image
FROM golang:1.17.5-alpine

# Set the working directory to /app
WORKDIR /app

# Copy the go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Download and cache the Go module dependencies
RUN go mod download

# Copy the rest of the application source code to the container
COPY . .

# Build the application binary
RUN go build -o app cmd/main.go

# Expose port 8080 for the app to listen on
EXPOSE 8080

# Start the application binary when the container starts
CMD ["./app"]
