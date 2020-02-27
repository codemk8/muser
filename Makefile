ifndef $(tag)
	tag=latest
endif

build: cmd/*.go
	go build -o bin/muser ./cmd/main.go

test: pkg/*/*.go
	go test -v github.com/codemk8/muser/pkg/...

clean:
	rm -rf bin/*

module:
	go mod vendor