FROM golang:1.25-alpine as builder
WORKDIR /app
COPY . .
RUN go mod tidy

RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o ./the-governor main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /code
COPY --from=builder /app/the-governor ./the-governor
# COPY governor-config.yaml .
COPY config.json .
RUN chmod +x /code/the-governor

# CREATE THE DIRECTORY STRUCTURE FOR THE VOLUMEMOUNT
RUN mkdir -p /home/argocd/cmp-server/config && \
    chown -R 999:999 /home/argocd

# Switch to the argocd user
USER 999