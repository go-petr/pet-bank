#!/bin/sh
set -e

echo "run db migration"
/usr/app/migrate -path /usr/app/configs/db/migration -database "$DB_SOURCE" -verbose up

echo "start the app"
exec "$@"