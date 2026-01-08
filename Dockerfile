FROM golang:1.25-alpine

# Install basic tools you might need inside the shell
RUN apk add --no-cache git bash build-base make ffmpeg

# Install golang-migrate (NO CURL REQUIRED)
RUN wget -qO- https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz | tar xz -C /usr/local/bin migrate && \
    chmod +x /usr/local/bin/migrate

WORKDIR /app

# The 'wait' process: keeps the container alive without doing anything
CMD ["tail", "-f", "/dev/null"]