$env:GOTENBERG_URL = "http://localhost:3000"
Write-Host "GOTENBERG_URL=$env:GOTENBERG_URL"
go clean -testcache
go test -v ./internal/pkg/gotenberg/... -run Integration 