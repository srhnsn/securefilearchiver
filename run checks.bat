go vet ./...

golint ./...

errcheck -ignore "os:Close" ./...

@pause