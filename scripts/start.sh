#!/bin/sh
set -e
# Run echo command with red color (\033[0;31m)
echo -e "\033[0;31mRUN MIGRATIONS" 
migrate -path /usr/app/configs/db/migration -database "$DB_SOURCE" -verbose up

echo -e "\033[0;31mSTART THE APP"
exec "$@"