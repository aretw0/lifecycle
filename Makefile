.PHONY: test vet serve-docs

# Run all tests
test:
	go test ./...

# Run vet tool in all files
vet:
	go vet ./...

# Run local Go documentation server (pkgsite)
serve-docs:
	go tool godoc -http=:6060
