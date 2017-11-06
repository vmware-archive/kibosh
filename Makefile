default: all

GO_PACKAGES = $$(go list ./... | grep -v vendor)
GO_FILES = $$(find . -name "*.go" | grep -v vendor | uniq)

do-build:
	go build -o kibosh ./main.go

unit-test:
	@go test ${GO_PACKAGES}

fmt:
	gofmt -s -l -w $(GO_FILES)

vet:
	@go vet ${GO_PACKAGES}

test: unit-test vet

run:
	VCAP_SERVICES='{"kubo-odb":[{"credentials":{"kubeconfig":{"apiVersion":"v1","clusters":[{"cluster":{"certificate-authority-data":"bXktZmFrZWNlcnQ="}}],"users":[{"user":{"token":"bXktZmFrZWNlcnQ="}}]}}}]}' \
	SERVICE_ID=123 \
	SECURITY_USER_NAME=admin \
	SECURITY_USER_PASSWORD=pass \
	go run main.go

all: fmt test do-build
