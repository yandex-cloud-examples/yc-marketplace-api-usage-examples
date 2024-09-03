# Start from the base Go image
FROM golang:1.22.5-alpine as build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source from the current directory to the Working Directory inside the container
COPY ./fake .

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Build the Go app
RUN GOOS=linux go build -o main .

# Start a new stage from scratch
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=build /app/main .
COPY --from=build /app/x509 /app/x509

# Set the PORT environment variable
ENV PORT=8080

# Expose port 8080 to the outside
EXPOSE 8080

# Command to run the executable
CMD ["./main"]