# bot-bca

Intitial setup

- Fill config.txt with your

Initial dependency

```bash
go mod tidy
```

Run locally with

```
go run cmd/server/main.go
```

How to build

```
GOOS=windows GOARCH=amd64 go build -o bin/klikbca-amd64-win.exe cmd/server/main.go
GOOS=linux GOARCH=amd64 go build -o bin/klikbca-amd64-linux cmd/server/main.go
```
