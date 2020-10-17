# Add command
# cobra add download

build:
	goreleaser --snapshot --skip-publish --rm-dist

test:
	docker run --rm -i -e POSTGRES_PASSWORD=secret cappa:latest
	#go test ./cmd/... -v --volume=${PWD}:/app -w /app

image:
	docker build -t cappa:latest -f Dockerfile.test .


