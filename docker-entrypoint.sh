#!/bin/sh

set -e

# Make sure DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "ERROR: DATABASE_URL is not set"
    exit 1
fi

# Extract the host and port using sed
DATABASE_HOST=$(echo "$DATABASE_URL" | grep -oE '://([^@/]+@)?([^:/]+)' | grep -oE '([^@/]+)$')
DATABASE_PORT=$(echo "$DATABASE_URL" | grep -oE ':[0-9]+/' | grep -oE '[0-9]+')

# Check if both HOST and PORT were extracted successfully
if [ -n "$DATABASE_HOST" ] && [ -n "$DATABASE_PORT" ]; then
    echo "INFO: Waiting for Postgres in $DATABASE_HOST:$DATABASE_PORT"
else
    echo "Could not extract HOST and PORT from DATABASE_URL"
    exit 1
fi

echo "INFO: Waiting for Postgres to start..."
while ! nc -z ${DATABASE_HOST} ${DATABASE_PORT}; do sleep 0.1; done
echo "INFO: Postgres is up"

# sleep one more second to prevent from "pq: the database system is starting up" issue
sleep 1

exec "$@"
