default: all

GO_PACKAGES = $$(go list ./... | grep -v vendor)
GO_FILES = $$(find . -name "*.go" | grep -v vendor | uniq)

build:
	go build -o kibosh ./main.go

unit-test:
	@go test ${GO_PACKAGES}

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
	SERVICE_ID=123 \
	SECURITY_USER_NAME=admin \
	SECURITY_USER_PASSWORD=pass \
	go run main.go

clean:
	rm -rf vendor
	rm -f kibosh
	rm -f Gopkg.lock

HAS_DEP := $(shell command -v dep;)
HAS_BINDATA := $(shell command -v go-bindata;)

.PHONY: bootstrap
bootstrap:
ifndef HAS_DEP
	go get -u github.com/golang/dep/cmd/dep
endif
ifndef HAS_BINDATA
	go get github.com/jteeuwen/go-bindata/...
endif
	dep ensure -v
	scripts/setup-apimachinery.sh

all: fmt test do-build
