#!/bin/sh
# Start Go backend in background
/app/uptime &

# Start Nginx in foreground
nginx -g "daemon off;"
