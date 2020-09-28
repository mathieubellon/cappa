# Add command
# cobra add download

build:
	goreleaser --snapshot --skip-publish --rm-dist

test:
	go test ./cmd/... -v
