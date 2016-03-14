NAME = $(shell awk -F\" '/^const Name/ { print $$2 }' main.go)
VERSION = $(shell awk -F\" '/^const Version/ { print $$2 }' cmd/version.go)
TESTDEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' $(shell glide novendor))

all: deps build

deps:
	glide install
	echo $(TESTDEPS) | xargs -n1 go get

updatedeps:
	glide update

build: deps
	@mkdir -p bin/
	go build -o bin/$(NAME)

test: deps
	go test $(shell glide novendor) -timeout=30s -parallel=4
	go vet $(shell glide novendor)

xcompile: deps test
	@rm -rf build/
	@mkdir -p build
	gox \
		-os="darwin" \
		-os="dragonfly" \
		-os="freebsd" \
		-os="linux" \
		-os="openbsd" \
		-os="solaris" \
		-os="windows" \
		-output="build/{{.Dir}}_$(VERSION)_{{.OS}}_{{.Arch}}/$(NAME)"

package: xcompile
	$(eval FILES := $(shell ls build))
	@mkdir -p build/tgz
	for f in $(FILES); do \
		(cd $(shell pwd)/build && tar -zcvf tgz/$$f.tar.gz $$f); \
		echo $$f; \
	done

.PHONY: all deps updatedeps build test xcompile package
