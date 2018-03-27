# Package:    sre/ipalloc
# Build:      docker
# Arch:       x86_64
# Maintainer: jcamou
# Type:       container

OWNER?=   sre
PROGRAM?= ipalloc
VERSION?= $(shell git rev-list HEAD --max-count=1 --abbrev-commit)

LDFLAGS=-ldflags "-X main.Service=${SERVICE} -X main.Version=${VERSION} -X main.Commit=${VERSION}"

all: dep test build

dep:
	dep ensure

test:
	go test ./...

build: dep
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -x -v ${LDFLAGS} -o ${PROGRAM}

docker: build
	docker build --no-cache -t ${PROGRAM}:${VERSION} .
	docker tag ${PROGRAM}:${VERSION} reg.domain.net:443/${OWNER}/${PROGRAM}:${VERSION}
	docker push reg.domain.net:443/${OWNER}/${PROGRAM}:${VERSION}

clean:
	go clean -x -v
	rm -rf vendor/
