# Add command
# cobra add download

# go run main.go download
APPNAME=cappa
darwin:
	GOOS=darwin GOARCH=amd64 go build -o ./releases/darwin_amd64/${APPNAME} ${LDFLAGS} *.go

windows:
	GOOS=windows GOARCH=amd64 go build -o ./releases/windows_amd64/${APPNAME}.exe ${LDFLAGS} *.go

release:
	goreleaser --rm-dist

releasefake:
	goreleaser release --skip-publish

test:
	go test ./cmd/... -v
