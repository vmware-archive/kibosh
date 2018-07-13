default: all

GO_PACKAGES = $$(go list ./... ./cmd/loader | grep -v vendor)
GO_FILES = $$(find . -name "*.go" | grep -v vendor | uniq)
VERSION = $$(cat tiller-version)

LDFLAGS="-X github.com/cf-platform-eng/kibosh/pkg/helm.tillerTag=$(VERSION)"

linux:
	GOOS=linux GOARCH=amd64 go build -o kibosh.linux -ldflags ${LDFLAGS} ./cmd/kibosh/main.go

mac:
	GOOS=darwin GOARCH=amd64 go build -o kibosh.darwin -ldflags ${LDFLAGS} ./cmd/kibosh/main.go

build: linux mac

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


unit-test:
	@go test -ldflags ${LDFLAGS} ${GO_PACKAGES}

fmt:
	gofmt -s -l -w $(GO_FILES)

vet:
	@go vet ${GO_PACKAGES}

test: unit-test vet

generate:
	#counterfeiter -o test/fake_kubernetes_client.go k8s.io/client-go/kubernetes.Interface
	# ^ requires having k8s.io/client-go checked out, see https://git.io/vFo28
	#sed -i '' 's/FakeInterface/FakeK8sInterface/g' test/fake_kubernetes_client.go
	go generate ./...

run:
	VCAP_SERVICES='{"kubo-odb":[{"credentials":{"kubeconfig":{"apiVersion":"v1","clusters":[{"cluster":{"certificate-authority-data":"bXktZmFrZWNlcnQ="}}],"users":[{"user":{"token":"bXktZmFrZWNlcnQ="}}]}}}]}' \
	SECURITY_USER_NAME=admin \
	SECURITY_USER_PASSWORD=pass \
	go run -ldflags ${LDFLAGS} main.go

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
	rm -rf vendor
	rm -rf Gopkg.lock
	dep ensure -v

all: fmt test build build-loader build-bazaar build-bazaar-cli
