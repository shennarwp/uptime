# Notes
1. Your `nginx.conf` only proxies `/api`, so `/swagger/` isn't reachable through the container's port `80` — only directly on the Go backend's `:8080`. Add location `/swagger { proxy_pass http://127.0.0.1:8080; }` to `nginx` if you want it exposed.
2. Regenerate after API changes with `go tool swag init -g cmd/uptime/main.go --output api`