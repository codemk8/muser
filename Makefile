ifndef $(tag)
	tag=latest
endif

build: cmd/*.go
	CGO_ENABLED=0 go build -o bin/muser ./cmd/main.go

test: pkg/*/*.go
	go test -v github.com/codemk8/muser/pkg/...

clean:
	rm -rf bin/*

module:
	go mod vendor
