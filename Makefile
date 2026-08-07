binary = yo

version ?= dev
ldflags = -ldflags "-X main.version=$(version)"

release:
	GOOS=windows GOARCH=amd64 go build $(ldflags) -o ./bin/$(binary)_windows_amd64
	GOOS=linux GOARCH=amd64 go build $(ldflags) -o ./bin/$(binary)_linux_amd64
	GOOS=darwin GOARCH=amd64 go build $(ldflags) -o ./bin/$(binary)_darwin_amd64
