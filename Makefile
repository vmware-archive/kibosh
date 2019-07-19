go default: all

GO_PACKAGES = $$(go list ./... ./cmd/loader | grep -v vendor)
GO_FILES = $$(find . -name "*.go" | grep -v vendor | uniq)

build-kibosh-linux:
	GOOS=linux GOARCH=amd64 go build -o kibosh.linux ./cmd/kibosh/main.go

build-kibosh-mac:
	GOOS=darwin GOARCH=amd64 go build -o kibosh.darwin ./cmd/kibosh/main.go

build-kibosh: build-kibosh-linux build-kibosh-mac

build-loader-linux:
	GOOS=linux GOARCH=amd64 go build -o loader.linux ./cmd/loader/main.go

build-loader-mac:
	GOOS=darwin GOARCH=amd64 go build -o loader.mac ./cmd/loader/main.go

build-loader: build-loader-linux build-loader-mac

build-bazaar-mac:
	GOOS=darwin GOARCH=amd64 go build -o bazaar.mac ./cmd/bazaarapi/main.go

build-bazaar-linux:
	GOOS=linux GOARCH=amd64 go build -o bazaar.linux ./cmd/bazaarapi/main.go

build-bazaar: build-bazaar-linux build-bazaar-mac

build-bazaar-cli-mac:
	GOOS=darwin GOARCH=amd64 go build -o bazaarcli.mac ./cmd/bazaarcli/main.go

build-bazaar-cli-linux:
	GOOS=linux GOARCH=amd64 go build -o bazaarcli.linux ./cmd/bazaarcli/main.go

build-bazaar-cli: build-bazaar-cli-mac build-bazaar-cli-linux

build-template-tester-mac:
	GOOS=darwin GOARCH=amd64 go build -o template-tester.mac ./cmd/templatetester/main.go

build-template-tester-linux:
	GOOS=linux GOARCH=amd64 go build -o template-tester.linux ./cmd/templatetester/main.go

build-template-tester: build-template-tester-mac build-template-tester-linux

unit-test:
	@go test ${GO_PACKAGES}

fmt:
	gofmt -s -l -w $(GO_FILES)

vet:
	@go vet ${GO_PACKAGES}

test: unit-test vet

generate:
	#counterfeiter -o pkg/test/fake_kubernetes_client.go k8s.io/client-go/kubernetes.Interface
	# ^ requires having k8s.io/client-go checked out, see https://git.io/vFo28
	#sed -i '' 's/FakeInterface/FakeK8sInterface/g' pkg/test/fake_kubernetes_client.go
	go generate ./...

cleandep:
	rm -rf vendor
	rm -f Gopkg.lock

HAS_DEP := $(shell command -v dep;)
HAS_BINDATA := $(shell command -v go-bindata;)

boostrap: bootstrap

.PHONY: bootstrap
bootstrap:
ifndef HAS_DEP
	go get -u github.com/golang/dep/cmd/dep
endif
ifndef HAS_BINDATA
	go get github.com/jteeuwen/go-bindata/...
endif
	dep ensure -v

all: fmt test build-kibosh build-loader build-bazaar build-bazaar-cli build-template-tester
quick: fmt test build-kibosh-mac build-loader-mac build-template-tester-mac
